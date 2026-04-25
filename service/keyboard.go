package service

import (
	"fmt"

	"github.com/honeok/subscription-bot/subscription"
	"github.com/honeok/subscription-bot/telegram"
)

func mainKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "添加订阅", CallbackData: "add"},
				{Text: "查看订阅", CallbackData: "subs"},
			},
			{
				{Text: "生成提醒", CallbackData: "report"},
			},
		},
	}
}

func cancelKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "取消", CallbackData: "cancel"},
			},
		},
	}
}

func confirmClearKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "确认清空", CallbackData: "clear:yes"},
				{Text: "返回", CallbackData: "clear:no"},
			},
		},
	}
}

func subscriptionKeyboard(items []subscription.Subscription) *telegram.InlineKeyboardMarkup {
	keyboard := [][]telegram.InlineKeyboardButton{}
	for _, item := range items {
		keyboard = append(keyboard, []telegram.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("删除 #%d", item.ID),
				CallbackData: fmt.Sprintf("delete:%d", item.ID),
			},
		})
	}

	keyboard = append(keyboard,
		[]telegram.InlineKeyboardButton{
			{Text: "添加订阅", CallbackData: "add"},
			{Text: "刷新", CallbackData: "subs"},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "生成提醒", CallbackData: "report"},
			{Text: "清空", CallbackData: "clear"},
		},
	)

	return &telegram.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}
