package groupurl

import (
	"bufio"
	"errors"
	"math/rand"
	"net/url"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	g, err := loadFixture("examples/test.urls")
	if err != nil {
		t.Fatal(err)
	}

	if len(g.trees) != 3 {
		t.Fatalf("expected 3 trees, got %d", len(g.trees))
	}

	u, err := url.Parse("https://example.com/thesaurus/spill-marlin-elaborate-washtub-nephew/index.html")
	if err != nil {
		t.Fatal(err)
	}
	path := g.SimplifyPath(u)
	if path != "/thesaurus/Words/index.html" {
		t.Fatalf("expected /thesaurus/Words/index.html, got %s", path)
	}

	u, err = url.Parse("https://example.com/random/PMFKQYGHBQWKZYBETZFWMWBTCBCCXJ")
	if err != nil {
		t.Fatal(err)
	}
	path = g.SimplifyPath(u)
	if path != "/random/Words" {
		t.Fatalf("expected /random/Words, got %s", path)
	}

	u, err = url.Parse("https://example.com/2013/11/20/unrest-growl-expansion-bullish-pediatric-shadiness-plus")
	if err != nil {
		t.Fatal(err)
	}
	path = g.SimplifyPath(u)
	if path != "/YYYY/MM/DD/Words" {
		t.Fatalf("expected /YYYY/MM/DD/Words, got %s", path)
	}
}

func loadFixture(path string) (Grouper, error) {
	file, err := os.Open(path)
	if err != nil {
		return Grouper{}, err
	}
	defer file.Close()

	var urls []*url.URL
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		u, err := url.Parse(scanner.Text())
		if err != nil {
			return Grouper{}, err
		}

		urls = append(urls, u)
	}
	if err := scanner.Err(); err != nil {
		return Grouper{}, err
	}

	rand.Shuffle(len(urls), func(i, j int) {
		urls[i], urls[j] = urls[j], urls[i]
	})

	g, err := New()
	if err != nil {
		return Grouper{}, err
	}
	for _, u := range urls {
		g.Add(u)
	}

	return g, nil
}

func TestNew(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if len(g.trees) != 0 {
		t.Fatalf("expected 0 trees, got %d", len(g.trees))
	}

	_, err = New(func(g *Grouper) error {
		return errors.New("test")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCaseInsensitiveStringCounter(t *testing.T) {
	c := newCaseInsensitiveStringCounter(3)
	c.add("test")
	c.add("Test")
	if c.get("test") != 2 {
		t.Fatalf("expected 2, got %d", c.get("test"))
	}

	c.add("test1")
	c.add("test1")
	c.add("test1")

	c.add("test3")

	if c.population() != 3 {
		t.Fatalf("expected 3, got %d", c.population())
	}

	elements := c.topN(2)
	if len(elements) != 2 {
		t.Fatalf("expected 2, got %d", len(elements))
	}
	if elements[0] != "test1" {
		t.Fatalf("expected test1, got %s", elements[0])
	}
	if elements[1] != "test" {
		t.Fatalf("expected test, got %s", elements[1])
	}

	// Verify cardinality limits works
	c.add("test4")
	c.add("test5")

	if c.population() != 4 {
		t.Fatalf("expected 4, got %d", c.population())
	}
	if c.get("test4") != 0 {
		t.Fatalf("expected 0, got %d", c.get("test4"))
	}
	if c.get("cardinality") != 2 {
		t.Fatalf("expected 2, got %d", c.get("cardinality"))
	}
}
