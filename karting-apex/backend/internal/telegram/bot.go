// Package telegram — минимальный клиент Telegram Bot API: вход и уведомления о бронях.
package telegram

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultAPIBase = "https://api.telegram.org"

// Update — входящее обновление вебхука: сообщение или нажатие кнопки под сообщением.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// CallbackQuery — нажатие кнопки под сообщением бота. Data приходит от клиента Telegram
// и может быть подделана: доверять можно только From (его подтверждает Telegram).
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type Message struct {
	MessageID int64    `json:"message_id"`
	From      *User    `json:"from"`
	Chat      Chat     `json:"chat"`
	Text      string   `json:"text"`
	Contact   *Contact `json:"contact"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	UserID      int64  `json:"user_id"`
}

// ReplyKeyboard — клавиатура с кнопкой «Поделиться номером».
type ReplyKeyboard struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	OneTimeKeyboard bool               `json:"one_time_keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
}

type KeyboardButton struct {
	Text           string `json:"text"`
	RequestContact bool   `json:"request_contact,omitempty"`
}

// InlineKeyboard — кнопки под сообщением.
type InlineKeyboard struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard"`
}

type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// RemoveKeyboard убирает клавиатуру после входа.
type RemoveKeyboard struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

// Client вызывает Bot API. Токен живёт только внутри: в ошибки и логи он не попадает —
// ошибки net/http содержат URL запроса, а в URL Bot API зашит токен.
type Client struct {
	token   string
	apiBase string
	http    *http.Client
}

func NewClient(token string) *Client {
	return &Client{token: token, apiBase: DefaultAPIBase, http: &http.Client{Timeout: 15 * time.Second}}
}

// WithAPIBase — адрес API для тестов.
func (c *Client) WithAPIBase(base string) *Client {
	c.apiBase = base
	return c
}

// WebhookSecret выводит секрет вебхука из токена: Telegram присылает его в заголовке,
// и отдельную переменную окружения заводить не нужно. Из секрета токен не восстановить.
func WebhookSecret(token string) string {
	sum := sha256.Sum256([]byte("apex-telegram-webhook:" + token))
	return hex.EncodeToString(sum[:24])
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

var ErrAPI = errors.New("telegram api error")

// APIError — отказ Bot API с кодом: по нему видно, стоит ли повторять отправку
// (429, 5xx) или нет (403 — человек заблокировал бота, 400 — чата нет).
type APIError struct {
	Method      string
	Code        int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s: %s", ErrAPI, e.Method, e.Description)
}

func (e *APIError) Is(target error) bool { return target == ErrAPI }

// Permanent — повтор той же отправки ничего не даст.
func (e *APIError) Permanent() bool {
	return e.Code == http.StatusBadRequest || e.Code == http.StatusForbidden
}

// Blocked — человек заблокировал бота или удалил аккаунт.
func (e *APIError) Blocked() bool { return e.Code == http.StatusForbidden }

func (c *Client) call(ctx context.Context, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram %s: encode request: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram %s: build request failed", method)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		// Не заворачиваем err: его текст содержит URL с токеном.
		return fmt.Errorf("telegram %s: request failed (network or timeout)", method)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("telegram %s: read response failed", method)
	}
	var decoded apiResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("telegram %s: http %d, undecodable response", method, resp.StatusCode)
	}
	if !decoded.OK {
		code := decoded.ErrorCode
		if code == 0 {
			code = resp.StatusCode
		}
		return &APIError{Method: method, Code: code, Description: decoded.Description}
	}
	if result != nil {
		if err := json.Unmarshal(decoded.Result, result); err != nil {
			return fmt.Errorf("telegram %s: decode result: %w", method, err)
		}
	}
	return nil
}

// GetMe возвращает имя бота (без @).
func (c *Client) GetMe(ctx context.Context) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	if err := c.call(ctx, "getMe", struct{}{}, &me); err != nil {
		return "", err
	}
	if me.Username == "" {
		return "", fmt.Errorf("%w: getMe: empty username", ErrAPI)
	}
	return me.Username, nil
}

// SetWebhook подключает вебхук. Принимаются сообщения и нажатия кнопок; накопившиеся за
// время простоя обновления отбрасываются — старые запросы входа всё равно истекли.
func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	return c.call(ctx, "setWebhook", map[string]any{
		"url":                  url,
		"secret_token":         secret,
		"allowed_updates":      []string{"message", "callback_query"},
		"drop_pending_updates": true,
	}, nil)
}

// AnswerCallbackQuery отвечает на нажатие кнопки: убирает «часики» на кнопке и показывает
// короткую всплывающую подсказку (text может быть пустым).
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	payload := map[string]any{"callback_query_id": callbackQueryID}
	if text != "" {
		payload["text"] = text
	}
	return c.call(ctx, "answerCallbackQuery", payload, nil)
}

// EditMessageReplyMarkup меняет кнопки под сообщением; пустая клавиатура их убирает.
func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64, markup InlineKeyboard) error {
	if markup.InlineKeyboard == nil {
		markup.InlineKeyboard = [][]InlineButton{}
	}
	return c.call(ctx, "editMessageReplyMarkup", map[string]any{
		"chat_id":      chatID,
		"message_id":   messageID,
		"reply_markup": markup,
	}, nil)
}

// BotCommand — пункт меню команд бота.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SetMyCommands задаёт меню команд, которое Telegram показывает у поля ввода.
func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	return c.call(ctx, "setMyCommands", map[string]any{"commands": commands}, nil)
}

// SendMessage отправляет текст без разметки (parse_mode не задан): имя пользователя
// или код в тексте не сломают сообщение спецсимволами.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, markup any) error {
	payload := map[string]any{"chat_id": chatID, "text": text}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	return c.call(ctx, "sendMessage", payload, nil)
}
