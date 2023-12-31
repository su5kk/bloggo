package main

import (
	"context"
	"github.com/mmcdole/gofeed"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AtomicDuration struct {
	d  time.Duration
	mx sync.Mutex
}

func (a *AtomicDuration) Set(d time.Duration) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.d = d
}

func (a *AtomicDuration) Duration() time.Duration {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.d
}

func (a *AtomicDuration) String() string {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.d.String()
}

type App struct {
	fetcher  *Fetcher
	repo     *RSSrepo
	bot      *tgbotapi.BotAPI
	chatID   int64
	feedList map[string]struct{}

	sent    chan *gofeed.Item
	fetched chan map[*gofeed.Feed]struct{}
	wg      sync.WaitGroup

	FetchDelay *AtomicDuration
	SendDelay  *AtomicDuration

	FeedItemsLimit int
}

func NewApp(bot *tgbotapi.BotAPI, chatID int64) *App {
	repo := NewRepo()
	return &App{
		fetcher:  NewFetcher(),
		repo:     repo,
		chatID:   chatID,
		bot:      bot,
		feedList: map[string]struct{}{},
		sent:     make(chan *gofeed.Item),
		fetched:  make(chan map[*gofeed.Feed]struct{}),
		wg:       sync.WaitGroup{},

		// can be set via commands
		FetchDelay:     &AtomicDuration{13 * time.Second, sync.Mutex{}},
		SendDelay:      &AtomicDuration{24 * time.Second, sync.Mutex{}},
		FeedItemsLimit: 10,
	}
}

func (a *App) Run(ctx context.Context) {
	a.wg.Add(4)

	// fetch feeds
	go func() {
		defer a.wg.Done()
		a.fetch(ctx)
	}()

	// insert feeds
	go func() {
		defer a.wg.Done()
		for feeds := range a.fetched {
			err := a.repo.Insert(feeds)
			if err != nil {
				log.Printf("Failed to insert: %v", err)
			}
		}
	}()

	// send messages
	go func() {
		defer a.wg.Done()
		a.fanOut(ctx)
	}()

	// mark as sent
	go func() {
		defer a.wg.Done()
		for item := range a.sent {
			err := a.repo.MarkAsSent(item)
			if err != nil {
				log.Printf("Failed to update: %v", err)
			}
		}
	}()

	a.wg.Wait()
}

func (a *App) fetch(ctx context.Context) {
	last := func(feeds []*gofeed.Feed) map[*gofeed.Feed]struct{} {
		feedsMap := make(map[*gofeed.Feed]struct{})
		for _, feed := range feeds {
			if len(feed.Items) > a.FeedItemsLimit {
				feed.Items = feed.Items[:a.FeedItemsLimit]
			}
			feedsMap[feed] = struct{}{}
		}
		return feedsMap
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("fetcher done")

			close(a.fetched)
			return
		case <-time.After(a.FetchDelay.Duration()):
			feeds := last(a.fetcher.FetchFeeds(feedURLs))
			log.Printf("feeds fetched: %v", len(feeds))

			a.fetched <- feeds
		}
	}
}

func (a *App) fanOut(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("fanout done")

			close(a.sent)
			return
		case <-time.After(a.SendDelay.Duration()):
			items := a.repo.Get()
			log.Printf("items to send: %v", len(items))

			for _, item := range items {
				_, err := a.bot.Send(tgbotapi.NewMessage(a.chatID, a.formatMessage(item)))
				if err != nil {
					log.Printf("Failed to send message: %v", err)
					continue
				}

				a.sent <- item
			}
		}
	}
}

func (a *App) formatMessage(item *gofeed.Item) string {
	text := strings.Builder{}
	text.WriteString(item.Title)
	text.WriteString("\n")
	text.WriteString(item.Link)
	return text.String()
}
