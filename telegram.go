package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type tgResponse struct {
	Ok          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

type tgMessage struct {
	MessageID int64 `json:"message_id"`
	Chat      struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"chat"`
	Text string `json:"text"`
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

// Telegram holds bot credentials and the chat to notify. The chat ID comes
// from TELEGRAM_CHAT_ID (survives restarts — recommended on Render) or is
// auto-discovered when the user messages the bot.
type Telegram struct {
	Token string

	mu          sync.RWMutex
	chatID      int64
	chatName    string
	chatPinned  bool // chat ID came from env, survives restarts
	botUsername string
	lastError   string
}

func NewTelegram(token, chatIDStr string) *Telegram {
	t := &Telegram{Token: strings.TrimSpace(token)}
	if id, err := strconv.ParseInt(strings.TrimSpace(chatIDStr), 10, 64); err == nil && id != 0 {
		t.chatID = id
		t.chatName = "(pinned via TELEGRAM_CHAT_ID)"
		t.chatPinned = true
	} else if saved, name, ok := loadSavedChat(); ok {
		t.chatID = saved
		t.chatName = name
	}
	return t
}

func (t *Telegram) Configured() bool { return t != nil && t.Token != "" }

func (t *Telegram) ChatID() int64 {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.chatID
}

func (t *Telegram) Ready() bool { return t.Configured() && t.ChatID() != 0 }

func (t *Telegram) setError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err != nil {
		t.lastError = err.Error()
	} else {
		t.lastError = ""
	}
}

func (t *Telegram) Status() map[string]any {
	s := map[string]any{
		"configured": t.Configured(),
		"connected":  t.Ready(),
	}
	if t.Configured() {
		t.mu.RLock()
		s["botUsername"] = t.botUsername
		s["chatName"] = t.chatName
		s["chatId"] = t.chatID
		s["chatPinned"] = t.chatPinned
		if t.lastError != "" {
			s["lastError"] = t.lastError
		}
		t.mu.RUnlock()
	}
	return s
}

func (t *Telegram) call(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	apiURL := "https://api.telegram.org/bot" + t.Token + "/" + method

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var tr tgResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("Telegram %s: HTTP %d: %s", method, resp.StatusCode, string(body))
	}
	if !tr.Ok {
		return nil, fmt.Errorf("Telegram %s error: %s", method, tr.Description)
	}
	return tr.Result, nil
}

func (t *Telegram) SendMessage(ctx context.Context, text string) error {
	if !t.Ready() {
		return fmt.Errorf("telegram not connected")
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	_, err := t.call(ctx, "sendMessage", form)
	t.setError(err)
	return err
}

// SendPhoto uploads a JPEG (e.g. a traffic camera frame) with a caption.
func (t *Telegram) SendPhoto(ctx context.Context, jpeg []byte, caption string) error {
	if !t.Ready() {
		return fmt.Errorf("telegram not connected")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(t.ChatID(), 10))
	if caption != "" {
		if len(caption) > 1024 {
			caption = caption[:1024]
		}
		_ = w.WriteField("caption", caption)
	}
	part, err := w.CreateFormFile("photo", "camera.jpg")
	if err != nil {
		return err
	}
	if _, err := part.Write(jpeg); err != nil {
		return err
	}
	_ = w.Close()

	apiURL := "https://api.telegram.org/bot" + t.Token + "/sendPhoto"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.setError(err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))

	var tr tgResponse
	if err := json.Unmarshal(body, &tr); err != nil || !tr.Ok {
		err = fmt.Errorf("Telegram sendPhoto error: HTTP %d %s", resp.StatusCode, string(body))
		t.setError(err)
		return err
	}
	t.setError(nil)
	return nil
}

// SendLiveLocation creates a location pin that can be moved for livePeriod
// seconds. Falls back to a 24h period if the "indefinite" magic value is
// rejected. Returns the message ID used for subsequent updates.
func (t *Telegram) SendLiveLocation(ctx context.Context, lat, lon float64, heading int, livePeriod int) (int64, error) {
	if !t.Ready() {
		return 0, fmt.Errorf("telegram not connected")
	}
	send := func(period int) (json.RawMessage, error) {
		form := url.Values{}
		form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
		form.Set("latitude", strconv.FormatFloat(lat, 'f', 6, 64))
		form.Set("longitude", strconv.FormatFloat(lon, 'f', 6, 64))
		form.Set("live_period", strconv.Itoa(period))
		if heading > 0 && heading <= 360 {
			form.Set("heading", strconv.Itoa(heading))
		}
		return t.call(ctx, "sendLocation", form)
	}

	res, err := send(livePeriod)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "live_period") {
		res, err = send(86400)
	}
	t.setError(err)
	if err != nil {
		return 0, err
	}
	var msg tgMessage
	if err := json.Unmarshal(res, &msg); err != nil {
		return 0, err
	}
	return msg.MessageID, nil
}

func (t *Telegram) EditLiveLocation(ctx context.Context, messageID int64, lat, lon float64, heading int) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	form.Set("latitude", strconv.FormatFloat(lat, 'f', 6, 64))
	form.Set("longitude", strconv.FormatFloat(lon, 'f', 6, 64))
	if heading > 0 && heading <= 360 {
		form.Set("heading", strconv.Itoa(heading))
	}
	_, err := t.call(ctx, "editMessageLiveLocation", form)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	t.setError(err)
	return err
}

func (t *Telegram) StopLiveLocation(ctx context.Context, messageID int64) {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	_, _ = t.call(ctx, "stopMessageLiveLocation", form)
}

// WebhookSecret derives a stable secret from the bot token so webhook
// requests can be authenticated without extra configuration.
func (t *Telegram) WebhookSecret() string {
	sum := sha256.Sum256([]byte("buswatch-webhook:" + t.Token))
	return hex.EncodeToString(sum[:16])
}

// Init fetches the bot identity and wires up updates. With a public URL
// (Render), it registers a webhook — crucial on free tiers, because an
// incoming webhook *wakes* a sleeping service, while long-polling dies with
// it. Without a public URL (local dev), it falls back to long-polling.
func (t *Telegram) Init(ctx context.Context, publicURL string) {
	if !t.Configured() {
		return
	}

	if res, err := t.call(ctx, "getMe", url.Values{}); err == nil {
		var me struct {
			Username string `json:"username"`
		}
		if json.Unmarshal(res, &me) == nil {
			t.mu.Lock()
			t.botUsername = me.Username
			t.mu.Unlock()
		}
	} else {
		t.setError(err)
		log.Printf("telegram: getMe failed (check TELEGRAM_BOT_TOKEN): %v", err)
		return
	}

	if publicURL != "" {
		form := url.Values{}
		form.Set("url", strings.TrimRight(publicURL, "/")+"/api/telegram/webhook")
		form.Set("secret_token", t.WebhookSecret())
		form.Set("allowed_updates", `["message"]`)
		if _, err := t.call(ctx, "setWebhook", form); err != nil {
			t.setError(err)
			log.Printf("telegram: setWebhook failed, falling back to polling: %v", err)
		} else {
			log.Printf("telegram: webhook registered at %s/api/telegram/webhook", publicURL)
			return
		}
	}

	_, _ = t.call(ctx, "deleteWebhook", url.Values{})
	go t.runUpdatePoller(ctx)
}

// ProcessUpdate handles one incoming update (from webhook or poller):
// the sender becomes the notification target, and /start gets a confirmation.
func (t *Telegram) ProcessUpdate(ctx context.Context, u tgUpdate) {
	if u.Message == nil || u.Message.Chat.ID == 0 {
		return
	}
	name := u.Message.Chat.FirstName
	if u.Message.Chat.Username != "" {
		name = "@" + u.Message.Chat.Username
	}

	t.mu.Lock()
	isNew := t.chatID != u.Message.Chat.ID
	t.chatID = u.Message.Chat.ID
	t.chatName = name
	pinned := t.chatPinned
	t.mu.Unlock()

	if isNew {
		saveChat(u.Message.Chat.ID, name)
		log.Printf("telegram: connected to chat %d (%s)", u.Message.Chat.ID, name)
	}
	if isNew || strings.HasPrefix(u.Message.Text, "/start") {
		msg := "✅ Connected to Thai Bus Watch!\n\nBus alerts will arrive in this chat. Open the web app to start tracking a bus."
		if !pinned {
			msg += fmt.Sprintf("\n\n💡 To keep this connection across server restarts, set the environment variable TELEGRAM_CHAT_ID=%d in your hosting dashboard (Render → Environment).", u.Message.Chat.ID)
		}
		_ = t.SendMessage(ctx, msg)
	}
}

// runUpdatePoller long-polls getUpdates — local development fallback only.
func (t *Telegram) runUpdatePoller(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		form := url.Values{}
		form.Set("timeout", "25")
		form.Set("offset", strconv.FormatInt(offset, 10))
		form.Set("allowed_updates", `["message"]`)

		res, err := t.call(ctx, "getUpdates", form)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(10 * time.Second)
			continue
		}

		var updates []tgUpdate
		if err := json.Unmarshal(res, &updates); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1
			t.ProcessUpdate(ctx, u)
		}
	}
}

func chatFile() string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return filepath.Join(dir, "telegram_chat.json")
	}
	return filepath.Join(os.TempDir(), "thai_bus_watch_chat.json")
}

func saveChat(id int64, name string) {
	b, _ := json.Marshal(map[string]any{"chat_id": id, "name": name})
	_ = os.WriteFile(chatFile(), b, 0o600)
}

func loadSavedChat() (int64, string, bool) {
	b, err := os.ReadFile(chatFile())
	if err != nil {
		return 0, "", false
	}
	var v struct {
		ChatID int64  `json:"chat_id"`
		Name   string `json:"name"`
	}
	if json.Unmarshal(b, &v) != nil || v.ChatID == 0 {
		return 0, "", false
	}
	return v.ChatID, v.Name, true
}
