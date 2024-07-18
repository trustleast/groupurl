# groupurl

groupurl is a zero-dependency package to turn high cardinality URLs into lower cardinality patterns.
This is useful for collecting custom metrics without being high in cardinality which can lead to cost explosions.
It is also useful for analyzing the structure of a site to extract interesting URLs given a sitemap.

The package works in a stream based fashion so it works on receiving live data or processing offline.

Groupers are not thread safe.

## Examples

```bash
go run examples/file/main.go examples/test.urls $(head -n20 examples/test.urls)
```

This will print out a simple representation of the URLs in a given file.
If you supply any extra URLs after the URL file, it will print a simple representation of that URL.
