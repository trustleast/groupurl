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
