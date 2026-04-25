package subscription

import (
	"errors"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"
)

const DateLayout = "2006-01-02"

type Subscription struct {
	ID         int64
	UserID     int64
	ChatID     int64
	Name       string
	TargetDate time.Time
	CreatedAt  time.Time
}

func New(userID int64, chatID int64, name string, targetDate time.Time) (Subscription, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Subscription{}, errors.New("name is required")
	}
	if len([]rune(name)) > 80 {
		return Subscription{}, errors.New("name is too long")
	}

	return Subscription{
		UserID:     userID,
		ChatID:     chatID,
		Name:       name,
		TargetDate: normalizeDate(targetDate),
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func ParseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "/", "-"))
	if value == "" {
		return time.Time{}, errors.New("date is required")
	}

	targetDate, err := time.ParseInLocation(DateLayout, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must be YYYY-MM-DD")
	}

	return normalizeDate(targetDate), nil
}

func BuildReport(subscriptions []Subscription, now time.Time) string {
	if len(subscriptions) == 0 {
		return "当前没有订阅。发送 /add 开始添加。"
	}

	now = normalizeDate(now)
	items := append([]Subscription(nil), subscriptions...)
	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].TargetDate.Equal(items[j].TargetDate) {
			return items[i].ID < items[j].ID
		}
		return items[i].TargetDate.Before(items[j].TargetDate)
	})

	var builder strings.Builder
	builder.WriteString("<b>订阅提醒</b>\n")
	builder.WriteString("日期：")
	builder.WriteString(now.Format(DateLayout))
	builder.WriteString("\n\n")

	for index, item := range items {
		builder.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", index+1, html.EscapeString(item.Name)))
		builder.WriteString("   到期：")
		builder.WriteString(item.TargetDate.Format(DateLayout))
		builder.WriteString("\n")
		builder.WriteString("   状态：")
		builder.WriteString(html.EscapeString(StatusText(item.TargetDate, now)))
		if item.ID > 0 {
			builder.WriteString(fmt.Sprintf("\n   编号：#%d", item.ID))
		}
		if index != len(items)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

func StatusText(targetDate time.Time, now time.Time) string {
	targetDate = normalizeDate(targetDate)
	now = normalizeDate(now)
	days := int(targetDate.Sub(now).Hours() / 24)

	switch {
	case days < 0:
		return fmt.Sprintf("已过期 %d 天", -days)
	case days == 0:
		return "今天到期"
	case days <= 3:
		return fmt.Sprintf("还有 %d 天，需要关注", days)
	default:
		return fmt.Sprintf("还有 %d 天", days)
	}
}

func normalizeDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
