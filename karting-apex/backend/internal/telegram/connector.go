package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// Commands — меню команд бота. /start не показываем: вход начинается из приложения.
var Commands = []BotCommand{
	{Command: "stop", Description: "Отключить уведомления о бронях"},
	{Command: "notify", Description: "Включить уведомления о бронях"},
}

// Connector подключает бота в фоне: узнаёт его имя и регистрирует вебхук. В фоне —
// чтобы недоступный Telegram не задерживал старт сервиса; до успешного подключения
// Username пуст, и приложение просто не показывает кнопку Telegram.
type Connector struct {
	client     *Client
	webhookURL string
	secret     string
	logger     *slog.Logger
	username   atomic.Pointer[string]
}

// NewConnector: publicURL — внешний адрес сервиса (на Render — RENDER_EXTERNAL_URL).
// Без него вебхук не регистрируется, а бот не подключается: принимать обновления
// было бы неоткуда.
func NewConnector(client *Client, publicURL, secret string, logger *slog.Logger) *Connector {
	if logger == nil {
		logger = slog.Default()
	}
	c := &Connector{client: client, secret: secret, logger: logger}
	if publicURL != "" {
		c.webhookURL = strings.TrimRight(publicURL, "/") + "/telegram/webhook"
	}
	return c
}

// Username — имя бота без @ или пустая строка, пока бот не подключён.
func (c *Connector) Username() string {
	if value := c.username.Load(); value != nil {
		return *value
	}
	return ""
}

// Run пытается подключить бота, пока не получится, с растущей паузой (до 5 минут).
func (c *Connector) Run(ctx context.Context) {
	if c.webhookURL == "" {
		c.logger.Error("telegram login disabled: public URL is unknown (set RENDER_EXTERNAL_URL or PUBLIC_URL)")
		return
	}
	delay := 5 * time.Second
	for {
		if err := c.connect(ctx); err == nil {
			return
		} else if ctx.Err() != nil {
			return
		} else {
			c.logger.Error("telegram bot connect failed, will retry", "error", err, "retry_in", delay.String())
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		if delay < 5*time.Minute {
			delay *= 2
		}
	}
}

func (c *Connector) connect(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	username, err := c.client.GetMe(callCtx)
	if err != nil {
		return err
	}
	if err := c.client.SetWebhook(callCtx, c.webhookURL, c.secret); err != nil {
		return err
	}
	c.username.Store(&username)
	c.logger.Info("telegram login enabled", "bot", "@"+username)
	// Меню команд — удобство, а не условие работы: ошибку только пишем в журнал.
	if err := c.client.SetMyCommands(callCtx, Commands); err != nil {
		c.logger.Warn("telegram set bot commands failed", "error", err)
	}
	return nil
}
