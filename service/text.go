package service

import (
	"fmt"
	"html"
	"strings"

	"github.com/honeok/subscription-bot/subscription"
)

func helpText() string {
	return `<b>订阅提醒机器人</b>

常用命令：
/add 添加订阅
/addsub 名称 YYYY-MM-DD 快速添加
/subs 查看和管理订阅
/today 立即生成提醒
/delete 编号 删除订阅
/clear 清空当前会话订阅

示例：
<code>/addsub Netflix 2026-05-01</code>`
}

func addPromptText() string {
	return `发送订阅名称和日期即可添加。

格式：
<code>名称 YYYY-MM-DD</code>

示例：
<code>Netflix 2026-05-01</code>`
}

func subscriptionListText(items []subscription.Subscription) string {
	if len(items) == 0 {
		return "当前没有订阅。发送 /add 开始添加。"
	}

	var builder strings.Builder
	builder.WriteString("<b>当前订阅</b>\n\n")
	for index, item := range items {
		builder.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", index+1, html.EscapeString(item.Name)))
		builder.WriteString("   到期：")
		builder.WriteString(item.TargetDate.Format(subscription.DateLayout))
		builder.WriteString("\n")
		builder.WriteString("   编号：")
		builder.WriteString(fmt.Sprintf("#%d", item.ID))
		if index != len(items)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}
