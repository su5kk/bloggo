package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var feedURLs = map[string]struct{}{
	"https://blog.golang.org/feed.atom":            {},
	"https://www.cockroachlabs.com/blog/index.xml": {},
	"https://matklad.github.io/feed.xml":           {},
	"https://envoy.engineering/feed":               {},
	"https://eng.lyft.com/feed":                    {},
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handleSignals(cancel)

	apitoken := os.Getenv("TELEGRAM_TOKEN")
	if apitoken == "" {
		panic("TELEGRAM_TOKEN env var is not set")
	}

	chatid := os.Getenv("TELEGRAM_CHAT_ID")
	if chatid == "" {
		panic("TELEGRAM_CHAT_ID env var is not set")
	}

	bot, err := tgbotapi.NewBotAPI(apitoken)
	if err != nil {
		panic(err)
	}

	chatID, err := strconv.ParseInt(chatid, 10, 64)
	if err != nil {
		panic(err)
	}

	a := NewApp(bot, chatID)

	go handleCommands(ctx, bot, a)

	log.Printf("Starting app")
	a.Run(ctx)
}

func handleCommands(ctx context.Context, bot *tgbotapi.BotAPI, a *App) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "hello")
			switch update.Message.Command() {
			case "setfd":
				newDelay, err := time.ParseDuration(update.Message.CommandArguments())
				if err != nil {
					// Send an error message back to the user
					msg.Text = "Error: " + err.Error()
					continue
				}

				a.FetchDelay.Set(newDelay)
				msg.Text = "Fetch delay set to " + update.Message.CommandArguments()
			case "setsd":
				newDelay, err := time.ParseDuration(update.Message.CommandArguments())
				if err != nil {
					// Send an error message back to the user
					msg.Text = "Error: " + err.Error()
					continue
				}

				a.SendDelay.Set(newDelay)
				msg.Text = "Send delay set to " + update.Message.CommandArguments()
			case "setlim":
				newLim, err := strconv.Atoi(update.Message.CommandArguments())
				if err != nil {
					// Send an error message back to the user
					msg.Text = "Error: " + err.Error()
					continue
				}
				a.FeedItemsLimit = newLim
				msg.Text = "Feed items limit set to " + update.Message.CommandArguments()
			case "config":
				msg.Text = "Fetch delay: " + a.FetchDelay.String() +
					"\nSend delay: " + a.SendDelay.String() +
					"\nFeed items limit: " + strconv.Itoa(a.FeedItemsLimit)
			case "help":
				msg.Text = "Available commands:\n/setfd <seconds>\n/setsd <seconds>\n/setlim <number>\n/config\n/help"
			}

			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
		}
	}
}

func handleSignals(cancel context.CancelFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigs
	log.Printf("Received signal: %v, shutting down...\n", sig)
	cancel()
}
