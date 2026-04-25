package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/honeok/subscription-bot/option"
	"github.com/honeok/subscription-bot/service"
	"github.com/honeok/subscription-bot/storage"
	"github.com/honeok/subscription-bot/telegram"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	config, err := option.Load()
	if err != nil {
		logger.Fatal(err)
	}

	store, err := storage.OpenSQLite(config.DataPath)
	if err != nil {
		logger.Fatal(err)
	}
	defer store.Close()

	client := telegram.NewClient(config.Token)
	bot := service.NewBot(config, client, store, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal(err)
	}
}
