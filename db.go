package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
)

type RSSrepo struct {
	db *sql.DB
}

const createStmt = `
CREATE TABLE IF NOT EXISTS rss (
	link text, 
	title text, 
	sent integer,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(link)
)
`

func NewRepo() *RSSrepo {
	// Check if the 'config' directory exists, create if not
	_, err := os.Stat("config")
	if os.IsNotExist(err) {
		err := os.Mkdir("config", 0755) // Create the directory with necessary permissions
		if err != nil {
			panic(err)
		}
	}

	dsn := "file:config/rss.db?cache=shared"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(createStmt)
	if err != nil {
		panic(err)
	}
	return &RSSrepo{db: db}
}

const insertStmt = `
INSERT OR IGNORE INTO rss (link, title, sent, created_at) 
VALUES (?, ?, ?, ?)
`

func (r *RSSrepo) Insert(feeds map[*gofeed.Feed]struct{}) error {
	for feed := range feeds {
		for _, item := range feed.Items {
			_, err := r.db.Exec(insertStmt, item.Link, item.Title, 0, time.Now())
			if err != nil {
				log.Printf("Failed to insert: %v", err)
				return err
			}
		}
	}
	return nil
}

const selectStmt = `
SELECT title, link 
FROM rss 
WHERE sent = 0
ORDER BY created_at DESC 
LIMIT 10
`

func (r *RSSrepo) Get() []*gofeed.Item {
	rows, err := r.db.Query(selectStmt)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var items []*gofeed.Item
	for rows.Next() {
		item := &gofeed.Item{}
		err := rows.Scan(&item.Title, &item.Link)
		if err != nil {
			panic(err)
		}

		items = append(items, item)
	}

	return items
}

const markAsSentStmt = `
UPDATE rss
SET sent = 1
WHERE link = ?
`

func (r *RSSrepo) MarkAsSent(items ...*gofeed.Item) error {
	for _, item := range items {
		_, err := r.db.Exec(markAsSentStmt, item.Link)
		if err != nil {
			log.Printf("Failed to mark as sent: %v", err)
			return err
		}
	}
	return nil
}
