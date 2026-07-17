package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type convStep int

const (
	stepNone convStep = iota
	stepHost
	stepPort
	stepUser
	stepPass
)

type conversation struct {
	step convStep
	data map[string]string
}

type Bot struct {
	client   *tgbot.Bot
	key      []byte
	allowed  map[int64]bool
	sessions map[int64]*SSHClient
	convs    map[int64]*conversation
}

func NewBot(token string, key []byte, allowed []int64) *Bot {
	allowMap := make(map[int64]bool)
	for _, id := range allowed {
		allowMap[id] = true
	}

	b := &Bot{
		key:      key,
		allowed:  allowMap,
		sessions: make(map[int64]*SSHClient),
		convs:    make(map[int64]*conversation),
	}

	client, err := tgbot.New(token,
		tgbot.WithDefaultHandler(b.onMessage),
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	b.client = client

	b.registerCommands()
	return b
}

func (b *Bot) Start() {
	log.Println("Bot is running...")
	b.client.Start(context.Background())
}

func (b *Bot) registerCommands() {
	ctx := context.Background()

	cmds := []models.BotCommand{
		{Command: "start", Description: "Show welcome message"},
		{Command: "connect", Description: "Connect to SSH server"},
		{Command: "reconnect", Description: "Reconnect to last server"},
		{Command: "disconnect", Description: "Disconnect current session"},
		{Command: "status", Description: "Show connection status"},
		{Command: "cancel", Description: "Cancel connection setup"},
		{Command: "help", Description: "Show help"},
	}
	b.client.SetMyCommands(ctx, &tgbot.SetMyCommandsParams{Commands: cmds})

	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.cmdStart)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypeExact, b.cmdStart)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/status", tgbot.MatchTypeExact, b.cmdStatus)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/cancel", tgbot.MatchTypeExact, b.cmdCancel)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/disconnect", tgbot.MatchTypeExact, b.cmdDisconnect)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/reconnect", tgbot.MatchTypeExact, b.cmdReconnect)
	b.client.RegisterHandler(tgbot.HandlerTypeMessageText, "/connect", tgbot.MatchTypeExact, b.cmdConnect)
}

func (b *Bot) auth(ctx context.Context, update *models.Update) bool {
	if update.Message == nil || update.Message.Chat.Type != models.ChatTypePrivate {
		return false
	}
	if len(b.allowed) > 0 && !b.allowed[update.Message.From.ID] {
		b.send(ctx, update.Message.Chat.ID, "Access denied.")
		return false
	}
	return true
}

func (b *Bot) send(ctx context.Context, chatID int64, text string) {
	b.client.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

// --- Commands ---

func (b *Bot) cmdStart(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	b.send(ctx, update.Message.Chat.ID,
		"<b>SSH Bot</b>\n\n"+
			"Execute commands on remote servers via Telegram.\n\n"+
			"<b>Commands:</b>\n"+
			"/connect \\- Connect to an SSH server\n"+
			"/reconnect \\- Reconnect to last saved server\n"+
			"/disconnect \\- Disconnect current session\n"+
			"/status \\- Show connection status\n"+
			"/cancel \\- Cancel connection setup\n"+
			"/help \\- Show this message\n\n"+
			"<i>All passwords are encrypted at rest.</i>")
}

func (b *Bot) cmdStatus(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	uid := update.Message.From.ID
	chatID := update.Message.Chat.ID

	if s, ok := b.sessions[uid]; ok && s.connected {
		sess := getSession(uid)
		if sess != nil {
			b.send(ctx, chatID,
				fmt.Sprintf("\U0001F7E2 <b>Connected</b>\n<code>%s@%s:%d</code>",
					sess.Username, sess.Host, sess.Port))
			return
		}
		b.send(ctx, chatID, "\U0001F7E2 <b>Connected</b>")
		return
	}

	sess := getSession(uid)
	if sess != nil {
		b.send(ctx, chatID,
			fmt.Sprintf("\U0001F534 <b>Disconnected</b>\nLast: <code>%s@%s:%d</code>\nUse /reconnect",
				sess.Username, sess.Host, sess.Port))
		return
	}
	b.send(ctx, chatID, "\U0001F534 <b>No sessions</b>\nUse /connect to start")
}

func (b *Bot) cmdCancel(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	uid := update.Message.From.ID
	if _, ok := b.convs[uid]; ok {
		delete(b.convs, uid)
		b.send(ctx, update.Message.Chat.ID, "Connection setup cancelled.")
		return
	}
	b.send(ctx, update.Message.Chat.ID, "Nothing to cancel.")
}

func (b *Bot) cmdDisconnect(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	uid := update.Message.From.ID
	if s, ok := b.sessions[uid]; ok {
		s.Disconnect()
		delete(b.sessions, uid)
		b.send(ctx, update.Message.Chat.ID, "\U0001F50C Disconnected.")
		return
	}
	b.send(ctx, update.Message.Chat.ID, "No active connection.")
}

func (b *Bot) cmdReconnect(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	uid := update.Message.From.ID
	chatID := update.Message.Chat.ID

	sess := getSession(uid)
	if sess == nil {
		b.send(ctx, chatID, "No saved session. Use /connect first.")
		return
	}

	// Disconnect existing
	if s, ok := b.sessions[uid]; ok {
		s.Disconnect()
		delete(b.sessions, uid)
	}

	pw, err := Decrypt(sess.Password, b.key)
	if err != nil {
		b.send(ctx, chatID, "Failed to decrypt saved password.")
		return
	}

	b.send(ctx, chatID, fmt.Sprintf("\u23F3 Connecting to <code>%s@%s:%d</code>...", sess.Username, sess.Host, sess.Port))

	client := &SSHClient{}
	if err := client.Connect(sess.Host, sess.Port, sess.Username, pw); err != nil {
		b.send(ctx, chatID, fmt.Sprintf("\u274C Connection failed: <i>%s</i>", err.Error()))
		return
	}

	b.sessions[uid] = client
	b.send(ctx, chatID,
		fmt.Sprintf("\u2705 <b>Connected</b>\n<code>%s@%s:%d</code>\n\nSend any message to execute commands.\nType /disconnect to end session.",
			sess.Username, sess.Host, sess.Port))
}

func (b *Bot) cmdConnect(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if !b.auth(ctx, update) {
		return
	}
	uid := update.Message.From.ID
	chatID := update.Message.Chat.ID

	// Disconnect existing
	if s, ok := b.sessions[uid]; ok {
		s.Disconnect()
		delete(b.sessions, uid)
	}

	b.convs[uid] = &conversation{step: stepHost, data: make(map[string]string)}
	b.send(ctx, chatID, "Enter SSH host:")
}

// --- Message handler ---

func (b *Bot) onMessage(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	if !b.auth(ctx, update) {
		return
	}

	uid := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// Conversation flow
	if conv, ok := b.convs[uid]; ok {
		b.handleConv(ctx, uid, chatID, update.Message.ID, text, conv)
		return
	}

	// SSH command execution
	s, ok := b.sessions[uid]
	if !ok || !s.connected {
		return
	}

	start := time.Now()
	output, err := s.Exec(text)
	elapsed := time.Since(start).Milliseconds()

	sess := getSession(uid)
	serverInfo := ""
	if sess != nil {
		serverInfo = fmt.Sprintf("%s@%s:%d", sess.Username, sess.Host, sess.Port)
	}

	if err != nil {
		b.send(ctx, chatID, FormatError(text, err.Error(), serverInfo))
		return
	}

	b.send(ctx, chatID, FormatOutput(text, output, serverInfo, elapsed))
}

func (b *Bot) handleConv(ctx context.Context, uid int64, chatID int64, msgID int, text string, conv *conversation) {
	switch conv.step {
	case stepHost:
		host := strings.TrimSpace(text)
		if host == "" || len(host) > 255 {
			b.send(ctx, chatID, "Invalid host. Try again:")
			return
		}
		conv.data["host"] = host
		conv.step = stepPort
		b.send(ctx, chatID, "Port (default 22):")

	case stepPort:
		port := 22
		if strings.TrimSpace(text) != "" {
			p, err := strconv.Atoi(strings.TrimSpace(text))
			if err != nil || p < 1 || p > 65535 {
				b.send(ctx, chatID, "Invalid port. Enter a number 1\\-65535:")
				return
			}
			port = p
		}
		conv.data["port"] = strconv.Itoa(port)
		conv.step = stepUser
		b.send(ctx, chatID, "Username:")

	case stepUser:
		user := strings.TrimSpace(text)
		if user == "" {
			b.send(ctx, chatID, "Username cannot be empty. Try again:")
			return
		}
		conv.data["user"] = user
		conv.step = stepPass
		b.send(ctx, chatID, "Password:")

	case stepPass:
		pw := text
		conv.data["pass"] = pw

		// Delete password message
		b.client.DeleteMessage(ctx, &tgbot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: msgID,
		})

		host := conv.data["host"]
		port, _ := strconv.Atoi(conv.data["port"])
		user := conv.data["user"]
		delete(b.convs, uid)

		b.send(ctx, chatID, fmt.Sprintf("\u23F3 Connecting to <code>%s@%s:%d</code>...", user, host, port))

		client := &SSHClient{}
		if err := client.Connect(host, port, user, pw); err != nil {
			b.send(ctx, chatID, fmt.Sprintf("\u274C Connection failed: <i>%s</i>", err.Error()))
			return
		}

		b.sessions[uid] = client

		// Save encrypted session
		encPW, err := Encrypt(pw, b.key)
		if err != nil {
			b.send(ctx, chatID, "Connected but failed to save session.")
			return
		}
		saveSession(uid, host, port, user, encPW)

		b.send(ctx, chatID,
			fmt.Sprintf("\u2705 <b>Connected</b>\n<code>%s@%s:%d</code>\n\nSend any message to execute commands.\nType /disconnect to end session.",
				user, host, port))
	}
}
