package groupurl

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	regexYYYY         = regexp.MustCompile(`^\d{4}(/|$)`)
	regexWords        = regexp.MustCompile(`^([a-zA-Z0-9]+[-_]?){1,}(/|$)`)
	regexYYYYMMDD     = regexp.MustCompile(`^\d{4}/((0[1-9])|(1[0-2]))/((0[1-9])|([1-2][0-9])|(3[01]))(/|$)`)
	regexNumbers      = regexp.MustCompile(`^\d+(/|$)`)
	regexAlpha        = regexp.MustCompile(`^[a-zA-Z]+(/|$)`)
	regexAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9\-_. ]+(/|$)`)

	_yyyyEnd = int64(time.Now().Year())
)

const _yyyyStart = 1900

// Labels are a wrapper that Classifiers return to indicate how a path should be treated.
// This wrapper exists to allow the `NestedPathTokenClassifier` to specify a parent label.
// Custom implementations of Classifiers only need to specify `LabelFields`.
type Label struct {
	LabelFields
	parent LabelFields
}

func (l Label) parentOrSelf() LabelFields {
	if l.parent.Value != "" {
		return l.parent
	}
	return l.LabelFields
}

func (l Label) isZero() bool {
	return l.Value == ""
}

// LabelFields indicates how a label should be treated by the Grouper.
// Important implies that all fields should be preserved exactly and not grouped under a generic label.
// CardinalityLimit tells the grouper to record fields up to a certain limit, and then group the rest under a generic label.
// Value is the name of the label.
type LabelFields struct {
	Important        bool
	CardinalityLimit int
	Value            string
}

func (l LabelFields) cardinalityLimit() int {
	if l.CardinalityLimit == 0 && !l.Important {
		return -1
	}
	return l.CardinalityLimit
}

// PathTokenClassifier is an interface that defines a method to check if a prefix of a path matches a label.
// The prefix of the path the classifier matches should be returned along with the label.
// If there is no match, it is fine to return an empty Label{}.
// The match string is used to tell the Grouper how much of the path should be consumed by the classifier.
// It is suggested that a classifier consumes up the next '/' in the path or the rest of the path.
// You can look at regular expressions in classifier.go for examples of how this is done in regular expressions.
type PathTokenClassifier interface {
	Check(path string) (label Label, match string)
}

// RegexPathTokenClassifier is a classifier that uses a regular expression to match a token.
// If the token matches the regular expression, the classifier will return the specified label.
type RegexPathTokenClassifier struct {
	Regex *regexp.Regexp
	Label Label
}

func (r RegexPathTokenClassifier) Check(s string) (Label, string) {
	match := r.Regex.FindString(s)
	if match == "" {
		return Label{}, ""
	}
	return r.Label, match
}

// YearPathTokenClassifier is a classifier that matches a token that is a year between the specified start and end years.
// If the token is a year between the specified start and end years, the classifier will return a label with the value "YYYY".
type YearPathTokenClassifier struct {
	Start int64
	End   int64
}

func (y YearPathTokenClassifier) Check(s string) (Label, string) {
	match := regexYYYY.FindString(s)
	if match == "" {
		return Label{}, ""
	}
	num, err := strconv.ParseInt(match[:4], 10, 64)
	if err != nil {
		return Label{}, ""
	}
	if num >= y.Start && num <= y.End {
		return Label{
			LabelFields: LabelFields{
				Important: false,
				Value:     "YYYY",
			},
		}, match
	}
	return Label{}, ""
}

// NestedPathTokenClassifier indicates to the grouper that if multiple children classifiers are matched in a segment,
// the segment should be grouped under the parent.
// For example, assume you have a parent that is Letters and Numbers, and you have children that is either Letters or Numbers.
// If the grouper only sees Letters or Numbers, it will group the segment under that more specific Label.
// If it sees both, it will group the segment under the parent.
type NestedPathTokenClassifier struct {
	Parent   PathTokenClassifier
	Children []PathTokenClassifier
}

func (n NestedPathTokenClassifier) Check(s string) (Label, string) {
	label, match := n.Parent.Check(s)
	if label.isZero() {
		return Label{}, ""
	}

	for _, child := range n.Children {
		childLabel, _ := child.Check(match)
		if !childLabel.isZero() {
			return Label{
				parent:      label.LabelFields,
				LabelFields: childLabel.LabelFields,
			}, match
		}
	}

	return label, match
}

// YYYYMMDDClassifier returns a classifier that matches segments that is a date in the format YYYY/MM/DD.
func YYYYMMDDClassifier() RegexPathTokenClassifier {
	return RegexPathTokenClassifier{
		Regex: regexYYYYMMDD,
		Label: Label{
			LabelFields: LabelFields{
				Important: false,
				Value:     "YYYY/MM/DD",
			},
		},
	}
}

// AlphaNumericClassifier returns a classifier that matches segments that are alphanumeric or special characters.
func AlphaNumericClassifier() RegexPathTokenClassifier {
	return RegexPathTokenClassifier{
		Regex: regexAlphaNumeric,
		Label: Label{
			LabelFields: LabelFields{
				Important: false,
				Value:     "AlphaNumeric",
			},
		},
	}
}

// NumberClassifier returns a classifier that matches segments that are numeric.
func NumberClassifier() RegexPathTokenClassifier {
	return RegexPathTokenClassifier{
		Regex: regexNumbers,
		Label: Label{
			LabelFields: LabelFields{
				Important: false,
				Value:     "Number",
			},
		},
	}
}

// WordsClassifier returns a classifier that matches segments that words delimited by dashes.
func WordsClassifier() RegexPathTokenClassifier {
	return RegexPathTokenClassifier{
		Regex: regexWords,
		Label: Label{
			LabelFields: LabelFields{
				Important:        true,
				CardinalityLimit: 50,
				Value:            "Words",
			},
		},
	}
}

// LettersClassifier returns a classifier that matches segments that are letters.
func LettersClassifier() RegexPathTokenClassifier {
	return RegexPathTokenClassifier{
		Regex: regexAlpha,
		Label: Label{
			LabelFields: LabelFields{
				Important:        true,
				CardinalityLimit: 50,
				Value:            "Letters",
			},
		},
	}
}

func DefaultClassifiers() []PathTokenClassifier {
	return []PathTokenClassifier{
		YYYYMMDDClassifier(),
		YearPathTokenClassifier{
			Start: _yyyyStart,
			End:   _yyyyEnd,
		},
		NestedPathTokenClassifier{
			Parent: AlphaNumericClassifier(),
			Children: []PathTokenClassifier{
				NumberClassifier(),
				WordsClassifier(),
				LettersClassifier(),
			},
		},
	}
}

type pathToken struct {
	token string
	label Label
}

func labelPathTokens(path string, classifiers []PathTokenClassifier) []pathToken {
	var cleaned []pathToken
	for path != "" {
		if path[0] == '/' {
			path = path[1:]
			continue
		}

		label, match := labelPathToken(path, classifiers)
		if strings.HasPrefix(path, match) {
			cleaned = append(cleaned, pathToken{
				token: strings.TrimRight(match, "/"),
				label: label,
			})
			path = path[len(match):]
		} else {
			cleaned = append(cleaned, pathToken{
				token: path,
				label: Label{
					LabelFields: LabelFields{
						Important: false,
						Value:     "Unknown",
					},
				},
			})
			break
		}
	}

	return cleaned
}

func labelPathToken(path string, classifiers []PathTokenClassifier) (Label, string) {
	for _, classifier := range classifiers {
		if label, match := classifier.Check(path); !label.isZero() {
			return label, match
		}
	}
	return Label{
		LabelFields: LabelFields{
			Important: false,
			Value:     "Unknown",
		},
	}, path
}
