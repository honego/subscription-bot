package option

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token         string
	DataPath      string
	PushTime      string
	Location      *time.Location
	PollTimeout   time.Duration
	DefaultChatID int64
	AllowedUsers  map[int64]bool
}

func Load() (Config, error) {
	var config Config

	config.Token = firstEnv("TG_BOT_TOKEN", "TELEGRAM_BOT_TOKEN")
	if config.Token == "" {
		return config, errors.New("missing TG_BOT_TOKEN")
	}

	config.DataPath = firstEnv("DATA_PATH", "SUBSCRIPTION_BOT_DATA")
	if config.DataPath == "" {
		config.DataPath = defaultDataPath()
	}

	config.PushTime = firstEnv("PUSH_TIME", "DAILY_PUSH_TIME")
	if config.PushTime == "" {
		config.PushTime = "09:00"
	}
	if _, err := time.Parse("15:04", config.PushTime); err != nil {
		return config, fmt.Errorf("invalid PUSH_TIME %q: %w", config.PushTime, err)
	}

	locationName := os.Getenv("TZ")
	if locationName == "" {
		config.Location = time.Local
	} else {
		location, err := time.LoadLocation(locationName)
		if err != nil {
			return config, fmt.Errorf("invalid TZ %q: %w", locationName, err)
		}
		config.Location = location
	}

	config.PollTimeout = 45 * time.Second
	if value := os.Getenv("POLL_TIMEOUT"); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return config, fmt.Errorf("invalid POLL_TIMEOUT %q", value)
		}
		config.PollTimeout = time.Duration(seconds) * time.Second
	}

	config.AllowedUsers = make(map[int64]bool)
	if err := parseUserList(config.AllowedUsers, firstEnv("TG_USER_ID", "TG_ALLOWED_USER_IDS", "TELEGRAM_ALLOWED_USERS")); err != nil {
		return config, err
	}
	for userID := range config.AllowedUsers {
		if config.DefaultChatID == 0 {
			config.DefaultChatID = userID
		}
	}

	return config, nil
}

func (c Config) IsAllowed(userID int64) bool {
	return len(c.AllowedUsers) == 0 || c.AllowedUsers[userID]
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func defaultDataPath() string {
	return filepath.Join("data", "subscriptions.db")
}

func parseUserList(users map[int64]bool, value string) error {
	if value == "" {
		return nil
	}

	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	}) {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		userID, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user id %q: %w", field, err)
		}
		users[userID] = true
	}

	return nil
}
