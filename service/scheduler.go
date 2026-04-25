package service

import (
	"context"
	"log"
	"time"

	"github.com/honeok/subscription-bot/option"
	"github.com/honeok/subscription-bot/storage"
	"github.com/honeok/subscription-bot/subscription"
	"github.com/honeok/subscription-bot/telegram"
)

type Scheduler struct {
	config       option.Config
	client       *telegram.Client
	store        storage.Store
	logger       *log.Logger
	lastSentDate string
}

func NewScheduler(config option.Config, client *telegram.Client, store storage.Store, logger *log.Logger) *Scheduler {
	return &Scheduler{
		config: config,
		client: client,
		store:  store,
		logger: logger,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.tick(ctx, nowIn(s.config.Location)); err != nil {
				s.logger.Printf("scheduler: %v", err)
			}
		}
	}
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) error {
	today := now.Format(subscription.DateLayout)
	if s.lastSentDate == today {
		return nil
	}
	if !isAfterPushTime(now, s.config.PushTime) {
		return nil
	}

	items, err := s.store.ListAll(ctx)
	if err != nil {
		return err
	}

	byChat := make(map[int64][]subscription.Subscription)
	for _, item := range items {
		byChat[item.ChatID] = append(byChat[item.ChatID], item)
	}
	if len(byChat) == 0 && s.config.DefaultChatID != 0 {
		byChat[s.config.DefaultChatID] = nil
	}

	for chatID, chatItems := range byChat {
		if len(chatItems) == 0 {
			continue
		}
		if err := s.client.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:                chatID,
			Text:                  subscription.BuildReport(chatItems, now),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		}); err != nil {
			return err
		}
	}

	s.lastSentDate = today
	return nil
}

func isAfterPushTime(now time.Time, pushTime string) bool {
	parsed, err := time.Parse("15:04", pushTime)
	if err != nil {
		return false
	}

	nowMinute := now.Hour()*60 + now.Minute()
	pushMinute := parsed.Hour()*60 + parsed.Minute()
	return nowMinute >= pushMinute
}
