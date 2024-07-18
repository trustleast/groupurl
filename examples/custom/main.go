package main

import (
	"strings"

	"github.com/trustleast/groupurl"
)

const _mySpecialField = "foo/"

type CustomPathTokenClassifier struct{}

func (c CustomPathTokenClassifier) Check(path string) (groupurl.Label, string) {
	if strings.HasPrefix(path, _mySpecialField) {
		return groupurl.Label{
			LabelFields: groupurl.LabelFields{
				Important: true,
				Value:     "SpecialToken",
			},
		}, _mySpecialField
	}
	return groupurl.Label{}, ""
}

func main() {
	groupurl.New(groupurl.WithClassifiers([]groupurl.PathTokenClassifier{
		CustomPathTokenClassifier{},
	}))
}
