package storage

import (
	"context"

	"github.com/honeok/subscription-bot/subscription"
)

type Store interface {
	Add(ctx context.Context, item subscription.Subscription) (subscription.Subscription, error)
	ListByChat(ctx context.Context, chatID int64) ([]subscription.Subscription, error)
	ListAll(ctx context.Context) ([]subscription.Subscription, error)
	Delete(ctx context.Context, chatID int64, id int64) (bool, error)
	DeleteAll(ctx context.Context, chatID int64) (int64, error)
	Close() error
}
