package service

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/honeok/subscription-bot/constant"
	"github.com/honeok/subscription-bot/option"
	"github.com/honeok/subscription-bot/storage"
	"github.com/honeok/subscription-bot/subscription"
	"github.com/honeok/subscription-bot/telegram"
)

type Bot struct {
	config  option.Config
	client  *telegram.Client
	store   storage.Store
	logger  *log.Logger
	pending map[int64]string
}

func NewBot(config option.Config, client *telegram.Client, store storage.Store, logger *log.Logger) *Bot {
	return &Bot{
		config:  config,
		client:  client,
		store:   store,
		logger:  logger,
		pending: make(map[int64]string),
	}
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.registerCommands(ctx); err != nil {
		b.logger.Printf("register commands: %v", err)
	}

	scheduler := NewScheduler(b.config, b.client, b.store, b.logger)
	go scheduler.Run(ctx)

	b.logger.Printf("%s %s started", constant.Name, constant.Version)

	offset := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.client.GetUpdates(ctx, offset, b.config.PollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			b.logger.Printf("get updates: %v", err)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			if err := b.handleUpdate(ctx, update); err != nil {
				b.logger.Printf("handle update %d: %v", update.UpdateID, err)
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update telegram.Update) error {
	if update.Message != nil {
		return b.handleMessage(ctx, update.Message)
	}
	if update.CallbackQuery != nil {
		return b.handleCallback(ctx, update.CallbackQuery)
	}
	return nil
}

func (b *Bot) handleMessage(ctx context.Context, message *telegram.Message) error {
	if message.From == nil || message.Text == "" {
		return nil
	}
	if !b.config.IsAllowed(message.From.ID) {
		return b.sendText(ctx, message.Chat.ID, "未授权访问。", nil)
	}

	text := strings.TrimSpace(message.Text)
	if action := b.pending[message.From.ID]; action == "add" && !strings.HasPrefix(text, "/") {
		delete(b.pending, message.From.ID)
		return b.addSubscription(ctx, message.From.ID, message.Chat.ID, text)
	}

	command, args := splitCommand(text)
	switch command {
	case "start", "help":
		return b.sendText(ctx, message.Chat.ID, helpText(), mainKeyboard())
	case "add", "addsub":
		if args == "" {
			b.pending[message.From.ID] = "add"
			return b.sendText(ctx, message.Chat.ID, addPromptText(), cancelKeyboard())
		}
		return b.addSubscription(ctx, message.From.ID, message.Chat.ID, args)
	case "subs", "subscriptions", "list":
		return b.sendSubscriptions(ctx, message.Chat.ID)
	case "today", "report":
		return b.sendReport(ctx, message.Chat.ID)
	case "delete", "del", "rm":
		return b.deleteSubscriptionCommand(ctx, message.Chat.ID, args)
	case "clear", "removeall":
		return b.sendText(ctx, message.Chat.ID, "确认删除当前会话中的全部订阅？", confirmClearKeyboard())
	case "cancel":
		delete(b.pending, message.From.ID)
		return b.sendText(ctx, message.Chat.ID, "已取消当前操作。", mainKeyboard())
	default:
		return b.sendText(ctx, message.Chat.ID, helpText(), mainKeyboard())
	}
}

func (b *Bot) handleCallback(ctx context.Context, query *telegram.CallbackQuery) error {
	if !b.config.IsAllowed(query.From.ID) {
		return b.client.AnswerCallbackQuery(ctx, query.ID, "未授权")
	}
	if query.Message == nil {
		return b.client.AnswerCallbackQuery(ctx, query.ID, "")
	}

	chatID := query.Message.Chat.ID
	if err := b.client.AnswerCallbackQuery(ctx, query.ID, ""); err != nil {
		b.logger.Printf("answer callback: %v", err)
	}

	switch {
	case query.Data == "help":
		return b.editText(ctx, chatID, query.Message.MessageID, helpText(), mainKeyboard())
	case query.Data == "add":
		b.pending[query.From.ID] = "add"
		return b.sendText(ctx, chatID, addPromptText(), cancelKeyboard())
	case query.Data == "subs":
		return b.editSubscriptions(ctx, chatID, query.Message.MessageID)
	case query.Data == "report":
		return b.editReport(ctx, chatID, query.Message.MessageID)
	case query.Data == "clear":
		return b.editText(ctx, chatID, query.Message.MessageID, "确认删除当前会话中的全部订阅？", confirmClearKeyboard())
	case query.Data == "clear:no":
		return b.editSubscriptions(ctx, chatID, query.Message.MessageID)
	case query.Data == "clear:yes":
		count, err := b.store.DeleteAll(ctx, chatID)
		if err != nil {
			return err
		}
		return b.editText(ctx, chatID, query.Message.MessageID, fmt.Sprintf("已删除 %d 个订阅。", count), mainKeyboard())
	case query.Data == "cancel":
		delete(b.pending, query.From.ID)
		return b.editText(ctx, chatID, query.Message.MessageID, "已取消当前操作。", mainKeyboard())
	case strings.HasPrefix(query.Data, "delete:"):
		id, err := strconv.ParseInt(strings.TrimPrefix(query.Data, "delete:"), 10, 64)
		if err != nil {
			return err
		}
		ok, err := b.store.Delete(ctx, chatID, id)
		if err != nil {
			return err
		}
		if !ok {
			return b.editText(ctx, chatID, query.Message.MessageID, "没有找到这个订阅。", mainKeyboard())
		}
		return b.editSubscriptions(ctx, chatID, query.Message.MessageID)
	default:
		return b.editText(ctx, chatID, query.Message.MessageID, helpText(), mainKeyboard())
	}
}

func (b *Bot) addSubscription(ctx context.Context, userID int64, chatID int64, value string) error {
	name, dateValue, ok := splitAddArgs(value)
	if !ok {
		b.pending[userID] = "add"
		return b.sendText(ctx, chatID, addPromptText(), cancelKeyboard())
	}

	targetDate, err := subscription.ParseDate(dateValue)
	if err != nil {
		b.pending[userID] = "add"
		return b.sendText(ctx, chatID, "日期格式不对，请使用 YYYY-MM-DD。\n\n"+addPromptText(), cancelKeyboard())
	}

	item, err := subscription.New(userID, chatID, name, targetDate)
	if err != nil {
		b.pending[userID] = "add"
		return b.sendText(ctx, chatID, err.Error()+"。\n\n"+addPromptText(), cancelKeyboard())
	}

	item, err = b.store.Add(ctx, item)
	if err != nil {
		return err
	}

	text := fmt.Sprintf("已添加：<b>%s</b>\n到期：%s\n编号：#%d", html.EscapeString(item.Name), item.TargetDate.Format(subscription.DateLayout), item.ID)
	return b.sendText(ctx, chatID, text, mainKeyboard())
}

func (b *Bot) sendSubscriptions(ctx context.Context, chatID int64) error {
	items, err := b.store.ListByChat(ctx, chatID)
	if err != nil {
		return err
	}

	return b.sendText(ctx, chatID, subscriptionListText(items), subscriptionKeyboard(items))
}

func (b *Bot) editSubscriptions(ctx context.Context, chatID int64, messageID int) error {
	items, err := b.store.ListByChat(ctx, chatID)
	if err != nil {
		return err
	}

	return b.editText(ctx, chatID, messageID, subscriptionListText(items), subscriptionKeyboard(items))
}

func (b *Bot) sendReport(ctx context.Context, chatID int64) error {
	items, err := b.store.ListByChat(ctx, chatID)
	if err != nil {
		return err
	}

	return b.sendText(ctx, chatID, subscription.BuildReport(items, nowIn(b.config.Location)), mainKeyboard())
}

func (b *Bot) editReport(ctx context.Context, chatID int64, messageID int) error {
	items, err := b.store.ListByChat(ctx, chatID)
	if err != nil {
		return err
	}

	return b.editText(ctx, chatID, messageID, subscription.BuildReport(items, nowIn(b.config.Location)), mainKeyboard())
}

func (b *Bot) deleteSubscriptionCommand(ctx context.Context, chatID int64, args string) error {
	id, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(args), "#"), 10, 64)
	if err != nil || id <= 0 {
		return b.sendText(ctx, chatID, "用法：/delete <编号>，例如 /delete 3。", mainKeyboard())
	}

	ok, err := b.store.Delete(ctx, chatID, id)
	if err != nil {
		return err
	}
	if !ok {
		return b.sendText(ctx, chatID, "没有找到这个订阅。", mainKeyboard())
	}

	return b.sendText(ctx, chatID, "已删除。", mainKeyboard())
}

func (b *Bot) registerCommands(ctx context.Context) error {
	return b.client.SetMyCommands(ctx, []telegram.BotCommand{
		{Command: "start", Description: "打开主菜单"},
		{Command: "add", Description: "添加订阅"},
		{Command: "subs", Description: "查看和管理订阅"},
		{Command: "today", Description: "立即生成提醒"},
		{Command: "delete", Description: "删除指定订阅"},
		{Command: "clear", Description: "清空当前会话订阅"},
		{Command: "cancel", Description: "取消当前操作"},
	})
}

func (b *Bot) sendText(ctx context.Context, chatID int64, text string, keyboard *telegram.InlineKeyboardMarkup) error {
	return b.client.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		ReplyMarkup:           keyboard,
		DisableWebPagePreview: true,
	})
}

func (b *Bot) editText(ctx context.Context, chatID int64, messageID int, text string, keyboard *telegram.InlineKeyboardMarkup) error {
	return b.client.EditMessageText(ctx, telegram.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
}

func splitCommand(text string) (string, string) {
	if !strings.HasPrefix(text, "/") {
		return "", text
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}

	command := strings.TrimPrefix(fields[0], "/")
	if index := strings.IndexByte(command, '@'); index >= 0 {
		command = command[:index]
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, fields[0]))
	return strings.ToLower(command), args
}

func splitAddArgs(value string) (string, string, bool) {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return "", "", false
	}

	dateValue := fields[len(fields)-1]
	name := strings.TrimSpace(strings.TrimSuffix(value, dateValue))
	return name, dateValue, name != ""
}

func nowIn(location *time.Location) time.Time {
	if location == nil {
		return time.Now()
	}
	return time.Now().In(location)
}
