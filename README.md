# groupurl

[![Go build status](https://github.com/trustleast/groupurl/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/trustleast/groupurl/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/trustleast/groupurl)](https://goreportcard.com/report/github.com/trustleast/groupurl) [![Go Reference](https://pkg.go.dev/badge/github.com/trustleast/groupurl.svg)](https://pkg.go.dev/github.com/trustleast/groupurl)

groupurl is a zero-dependency package to turn high cardinality URLs into lower cardinality patterns.
This is useful for collecting custom metrics without being high in cardinality which can lead to cost explosions.
It is also useful for analyzing the structure of a site to extract interesting URLs given a sitemap.

The package works in a stream based fashion so it works on receiving live data or processing offline.

Groupers are not thread safe.

## Usage

Adding URLs and simplifying them
```go
package main

import (
	"fmt"
	"net/url"

	"github.com/trustleast/groupurl"
)

func main() {
	g, _ := groupurl.New()
	for i := 0; i < 100; i++ {
		u, _ := url.Parse(fmt.Sprintf("https://example.com/important-label/%d", i))
		g.Add(u)
	}
	u, _ := url.Parse("https://example.com/important-label/1")
	simplified := g.SimplifyPath(u)

	// Output: https://example.com/important-label/Number
	fmt.Println(simplified)
}
```

Adding custom classifiers
```go
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
```

## Examples

```bash
go run examples/file/main.go examples/test.urls $(head -n20 examples/test.urls)
```

This will print out a simple representation of the URLs in a given file.
If you supply any extra URLs after the URL file, it will print a simple representation of that URL.
