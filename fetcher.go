package main

import (
	"errors"
	"log"

	"github.com/mmcdole/gofeed"
)

type feedResult struct {
	feed *gofeed.Feed
	err  error
}

type Fetcher struct {
	parser *gofeed.Parser
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		parser: gofeed.NewParser(),
	}
}

func (f *Fetcher) FetchFeeds(urls map[string]struct{}) []*gofeed.Feed {
	results := make(chan feedResult, len(urls))

	for u := range urls {
		go f.fetchFeed(u, results)
	}

	var feeds []*gofeed.Feed
	for i := 0; i < len(urls); i++ {
		res := <-results
		if res.err != nil {
			log.Printf("Failed to fetch feed: %v", res.err)
			continue
		}

		feeds = append(feeds, res.feed)
	}

	return feeds
}

func (f *Fetcher) fetchFeed(url string, results chan feedResult) {
	feed, err := f.parser.ParseURL(url)
	if err != nil {
		err = errors.New("failed to parse " + url + ": " + err.Error())
		results <- feedResult{nil, err}
		return
	}

	results <- feedResult{feed, nil}
}
