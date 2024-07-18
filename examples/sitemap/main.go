package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"

	"github.com/trustleast/groupurl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sitemap <sitemap_url>")
		os.Exit(1)
	}

	g, err := groupurl.New()
	if err != nil {
		fmt.Println("Failed to initialize Grouper", err)
		os.Exit(1)
	}

	ctx := context.Background()
	u, err := url.Parse(os.Args[1])
	if err != nil {
		fmt.Println("Failed to parse URL", err)
		os.Exit(1)
	}

	urls, err := getSitemapURLs(ctx, u)
	if err != nil {
		fmt.Println("Failed to get URLs from sitemap", err)
		os.Exit(1)
	}

	for _, u := range urls {
		g.Add(u)
	}

	g.Print()

	for i := 0; i < 10 && i < len(urls); i++ {
		n := rand.Int() % len(urls)
		fmt.Println(urls[n], " -> ", g.SimplifyPath(urls[n]))
	}
}

type sitemapContainer struct {
	URL []struct {
		Location string `xml:"loc"`
	} `xml:"url,omitempty"`
}

func getSitemapURLs(ctx context.Context, u *url.URL) ([]*url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sc sitemapContainer
	if err = xml.NewDecoder(resp.Body).Decode(&sc); err != nil {
		return nil, err
	}

	var urls []*url.URL
	for _, u := range sc.URL {
		parsed, err := url.Parse(u.Location)
		if err != nil {
			return nil, err
		}
		urls = append(urls, parsed)
	}

	return urls, nil
}
