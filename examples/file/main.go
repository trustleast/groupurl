package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"

	"github.com/trustleast/groupurl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: file <path>")
		fmt.Println("Path should point to a file containing new line separated URLs")
		os.Exit(1)
	}

	g, err := groupurl.New()
	if err != nil {
		fmt.Println("Failed to build grouper", err)
		os.Exit(1)
	}

	err = getURLs(os.Args[1], func(u *url.URL) error {
		g.Add(u)
		return nil
	})
	if err != nil {
		fmt.Println("Error getting URLs", err)
		os.Exit(1)
	}

	g.Print()

	for _, rawURL := range os.Args[2:] {
		u, err := url.Parse(rawURL)
		if err != nil {
			fmt.Println("Failed to parse url:", err)
			os.Exit(1)
		}
		fmt.Println(u, " -> ", g.SimplifyPath(u))
	}
}

func getURLs(path string, f func(*url.URL) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		u, err := url.Parse(scanner.Text())
		if err != nil {
			return fmt.Errorf("failed to parse URL: %w", err)
		}

		if err := f(u); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	return nil
}
