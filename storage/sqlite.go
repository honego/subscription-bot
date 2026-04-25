package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/honeok/subscription-bot/subscription"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, errors.New("sqlite path is required")
	}

	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Add(ctx context.Context, item subscription.Subscription) (subscription.Subscription, error) {
	result, err := s.db.ExecContext(ctx, `
INSERT INTO subscriptions (user_id, chat_id, name, target_date, created_at)
VALUES (?, ?, ?, ?, ?)
`, item.UserID, item.ChatID, item.Name, item.TargetDate.Format(subscription.DateLayout), item.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return item, err
	}

	item.ID, err = result.LastInsertId()
	if err != nil {
		return item, err
	}

	return item, nil
}

func (s *SQLiteStore) ListByChat(ctx context.Context, chatID int64) ([]subscription.Subscription, error) {
	return s.list(ctx, "WHERE chat_id = ?", chatID)
}

func (s *SQLiteStore) ListAll(ctx context.Context) ([]subscription.Subscription, error) {
	return s.list(ctx, "")
}

func (s *SQLiteStore) Delete(ctx context.Context, chatID int64, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE chat_id = ? AND id = ?`, chatID, id)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (s *SQLiteStore) DeleteAll(ctx context.Context, chatID int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE chat_id = ?`, chatID)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS subscriptions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	chat_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	target_date TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_chat_target
	ON subscriptions (chat_id, target_date, id);
`)
	return err
}

func (s *SQLiteStore) list(ctx context.Context, where string, args ...any) ([]subscription.Subscription, error) {
	query := `
SELECT id, user_id, chat_id, name, target_date, created_at
FROM subscriptions
` + where + `
ORDER BY target_date ASC, id ASC
`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []subscription.Subscription
	for rows.Next() {
		var item subscription.Subscription
		var targetDate string
		var createdAt string
		if err := rows.Scan(&item.ID, &item.UserID, &item.ChatID, &item.Name, &targetDate, &createdAt); err != nil {
			return nil, err
		}

		item.TargetDate, err = time.ParseInLocation(subscription.DateLayout, targetDate, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("invalid target date in database: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("invalid created_at in database: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
