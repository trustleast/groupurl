package groupurl

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type (
	// Grouper is a struct that groups URLs based on their path components.
	// It is not safe for concurrent use.
	// It can only keep track of a single host at a time so callers are encouraged to create a new Grouper per host.
	// The memory utilization of the Grouper is proportional to the number of unique paths it has seen.
	// However, it is possible to bound this memory by using Classifiers that emit labels marked as not `Important`,
	// or with `CardinalityLimit` set.
	Grouper struct {
		classifiers []PathTokenClassifier
		trees       map[int]urlTree
	}

	Option func(*Grouper) error
)

// WithClassifiers sets the classifiers to be used by the Grouper.
// If not specified, `DefaultClassifiers` will be used instead.
func WithClassifiers(classifiers []PathTokenClassifier) Option {
	return func(g *Grouper) error {
		g.classifiers = classifiers
		return nil
	}
}

// New creates a new Grouper with the provided options.
func New(options ...Option) (Grouper, error) {
	g := Grouper{
		classifiers: DefaultClassifiers(),
		trees:       make(map[int]urlTree),
	}
	for _, option := range options {
		if err := option(&g); err != nil {
			return Grouper{}, err
		}
	}

	return g, nil
}

// Add adds a url to the internal trees to keep statistics on it
// Groupers do not keep track of hosts URLs are associated with so it is suggested you use a different
// Grouper per host.
func (g Grouper) Add(u *url.URL) {
	tokens := labelPathTokens(u.Path, g.classifiers)
	t := g.getTree(u)
	t.add(tokens)
}

// Simplify simplifies a URL replacing path components with tokens representing original values.
// In the case that some tokens are low cardinality, the original value will be preserved.
func (g Grouper) SimplifyPath(u *url.URL) string {
	tokens := labelPathTokens(u.Path, g.classifiers)
	t := g.getTree(u)
	replaced := t.path(tokens)
	return "/" + strings.Join(replaced, "/")
}

// Print prints the internal trees to stdout to imply a nesting structure.
func (g Grouper) Print() {
	for _, t := range g.trees {
		t.print()
	}
}

func (g Grouper) getTree(u *url.URL) urlTree {
	originalTokenCount := strings.Count(strings.TrimRight(strings.TrimLeft(u.Path, "/"), "/"), "/")
	t, ok := g.trees[originalTokenCount]
	if !ok {
		t = newURLTree()
		g.trees[originalTokenCount] = t
	}
	return t
}

type caseInsensitiveStringCounter struct {
	limit       int
	total       int
	tokenCounts map[string]int
}

func newCaseInsensitiveStringCounter(limit int) caseInsensitiveStringCounter {
	return caseInsensitiveStringCounter{
		limit:       limit,
		tokenCounts: make(map[string]int),
	}
}

func (c *caseInsensitiveStringCounter) add(s string) {
	key := strings.ToLower(s)
	if _, ok := c.tokenCounts[key]; ok || c.limit == 0 || len(c.tokenCounts) < c.limit {
		c.tokenCounts[key]++
	} else {
		c.tokenCounts["cardinality"]++
	}
	c.total++
}

func (c caseInsensitiveStringCounter) population() int {
	return len(c.tokenCounts)
}

func (c caseInsensitiveStringCounter) get(s string) int {
	return c.tokenCounts[strings.ToLower(s)]
}

func (c caseInsensitiveStringCounter) isSignificant(s string) bool {
	averageSizePerToken := float64(c.population()) / float64(c.total)
	tokenPerPopulation := float64(c.get(s)) / float64(c.total)
	return (len(c.tokenCounts) < c.limit || c.limit == 0) && (averageSizePerToken < 0.01 ||
		tokenPerPopulation > averageSizePerToken)
}

func (c caseInsensitiveStringCounter) topN(n int) []string {
	type cardinalityAndToken struct {
		count int
		token string
	}
	var cardinalityAndTokens []cardinalityAndToken
	for k, v := range c.tokenCounts {
		cardinalityAndTokens = append(cardinalityAndTokens, cardinalityAndToken{
			count: v,
			token: k,
		})
	}

	sort.Slice(cardinalityAndTokens, func(i, j int) bool {
		return cardinalityAndTokens[i].count > cardinalityAndTokens[j].count
	})

	topN := n
	if len(cardinalityAndTokens) < n {
		topN = len(cardinalityAndTokens)
	}

	return mapSlice(cardinalityAndTokens[:topN], func(v cardinalityAndToken) string {
		return v.token
	})
}

type urlTree struct {
	Root *urlNode
}

func newURLTree() urlTree {
	return urlTree{
		Root: newURLNode(LabelFields{}),
	}
}

func (t urlTree) print() {
	t.printNode(t.Root, 0)
}

func (t urlTree) printNode(node *urlNode, depth int) {
	for _, child := range node.children {
		indent := strings.Repeat("  ", depth)

		tokens := filterSlice(child.tokenCounts.topN(20), child.tokenCounts.isSignificant)
		if len(tokens) > 0 && child.specificLabel.Important {
			fmt.Printf("%s/%s: %v(%d)\n", indent, child.specificLabel.Value, tokens, child.tokenCounts.total)
		} else {
			fmt.Printf("%s/%s: (%d)\n", indent, child.specificLabel.Value, child.tokenCounts.total)
		}

		t.printNode(child, depth+1)
	}
}

// Written iteratively instead of recursively to avoid deep stacks as these URLs can come from external clients.
func (t urlTree) add(tokens []pathToken) {
	current := t.Root
	for _, token := range tokens {
		parent := token.label.parentOrSelf()
		child, ok := current.children[parent]
		if !ok {
			child = newURLNode(token.label.LabelFields)
			current.children[parent] = child
		}

		// If we've found a child with a different label than the current token, we should mark it as a parent
		// so they are grouped together. At this point we also need to update our counters to reflect the new
		// labeling.
		if child.specificLabel.Value != token.label.LabelFields.Value {
			child.specificLabel = parent
			child.tokenCounts.limit = parent.CardinalityLimit
		}

		child.tokenCounts.add(token.token)
		current = child
	}
}

func (t urlTree) path(tokens []pathToken) []string {
	var replaced []string
	current := t.Root
	for idx, token := range tokens {
		parent := token.label.parentOrSelf()
		child, ok := current.children[parent]
		if !ok {
			return append(replaced, mapSlice(tokens[idx:], func(v pathToken) string {
				return v.token
			})...)
		}
		if child.specificLabel.Important && child.tokenCounts.isSignificant(token.token) {
			replaced = append(replaced, token.token)
		} else {
			replaced = append(replaced, child.specificLabel.Value)
		}

		current = child
	}
	return replaced
}

type urlNode struct {
	specificLabel LabelFields
	children      map[LabelFields]*urlNode
	tokenCounts   caseInsensitiveStringCounter
}

func newURLNode(label LabelFields) *urlNode {
	return &urlNode{
		specificLabel: label,
		children:      make(map[LabelFields]*urlNode),
		tokenCounts:   newCaseInsensitiveStringCounter(label.cardinalityLimit()),
	}
}

func mapSlice[X any, Y any](in []X, f func(X) Y) []Y {
	var result []Y
	for _, v := range in {
		result = append(result, f(v))
	}
	return result
}

func filterSlice[X any](in []X, f func(X) bool) []X {
	var result []X
	for _, v := range in {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}
