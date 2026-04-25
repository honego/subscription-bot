# subscription-bot

Go 版本的 Telegram 订阅到期提醒机器人。

这个项目按下面三个方向做第一版：

- 功能对齐 `tslcat/subscription-manager-bot`：添加目标日期、查看订阅、每日推送、临近到期高亮、SQLite 持久化。
- 交互借鉴 `Rongronggg9/RSS-to-Telegram-Bot`：命令清晰、按钮可管理、私聊/群组都能用、部署参数少。
- Go 工程规范参考 `SagerNet/sing-box`：小包拆分、`gofmt`、`go vet`。

## 快速开始

```bash
go mod tidy
go build -o subscription-bot ./cmd/subscription-bot
./subscription-bot
```

最小配置只需要机器人 Token：

```env
TG_BOT_TOKEN=123456:telegram-bot-token
```

常用可选配置：

```env
TG_USER_ID=123456789
TZ=Asia/Shanghai
PUSH_TIME=09:00
POLL_TIMEOUT=45
```

`TG_USER_ID` 设置后只有这些用户可以操作机器人；不设置则允许所有用户使用。多个用户可以用逗号分隔。
`TZ` 控制定时推送使用的时区，未设置时使用系统本地时区。`PUSH_TIME` 是每日推送时间，格式为 `HH:MM`，默认 `09:00`。

数据库默认放在当前工作目录的 `data/subscriptions.db`。如果文件已经存在，程序会直接打开它，不会覆盖重建。

## 命令

- `/start` 打开主菜单
- `/add` 进入添加流程
- `/addsub 名称 YYYY-MM-DD` 快速添加
- `/subs` 查看和管理订阅
- `/today` 立即生成提醒
- `/delete 编号` 删除订阅
- `/clear` 清空当前会话订阅
- `/cancel` 取消当前操作

## Docker

```bash
docker compose up -d --build
```

Docker Compose 会把本机 `./data` 挂载到容器的 `/app/data`，数据库会保存在 `./data/subscriptions.db`。
