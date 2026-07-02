package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// Telegram holds bot credentials and the discovered chat to notify.
// The chat ID comes from TELEGRAM_CHAT_ID or is auto-discovered when
// the user sends any message to the bot.
type Telegram struct {
	Token string

	mu          sync.RWMutex
	chatID      int64
	chatName    string
	botUsername string
}

func NewTelegram(token, chatIDStr string) *Telegram {
	t := &Telegram{Token: strings.TrimSpace(token)}
	if id, err := strconv.ParseInt(strings.TrimSpace(chatIDStr), 10, 64); err == nil && id != 0 {
		t.chatID = id
		t.chatName = "(from TELEGRAM_CHAT_ID)"
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

func (t *Telegram) Status() map[string]any {
	s := map[string]any{
		"configured": t.Configured(),
		"connected":  t.Ready(),
	}
	if t.Configured() {
		t.mu.RLock()
		s["botUsername"] = t.botUsername
		s["chatName"] = t.chatName
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
	return err
}

// SendLiveLocation creates a location pin that can be moved for livePeriod seconds.
// Returns the message ID used for subsequent updates.
func (t *Telegram) SendLiveLocation(ctx context.Context, lat, lon float64, heading int, livePeriod int) (int64, error) {
	if !t.Ready() {
		return 0, fmt.Errorf("telegram not connected")
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
	form.Set("latitude", strconv.FormatFloat(lat, 'f', 6, 64))
	form.Set("longitude", strconv.FormatFloat(lon, 'f', 6, 64))
	form.Set("live_period", strconv.Itoa(livePeriod))
	if heading > 0 && heading <= 360 {
		form.Set("heading", strconv.Itoa(heading))
	}
	res, err := t.call(ctx, "sendLocation", form)
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
	return err
}

func (t *Telegram) StopLiveLocation(ctx context.Context, messageID int64) {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(t.ChatID(), 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	_, _ = t.call(ctx, "stopMessageLiveLocation", form)
}

// RunUpdatePoller long-polls getUpdates so that the first person to message
// the bot becomes the notification target. It replies to confirm the link.
func (t *Telegram) RunUpdatePoller(ctx context.Context) {
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
		log.Printf("telegram: getMe failed (check TELEGRAM_BOT_TOKEN): %v", err)
	}

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
			if u.Message == nil || u.Message.Chat.ID == 0 {
				continue
			}
			name := u.Message.Chat.FirstName
			if u.Message.Chat.Username != "" {
				name = "@" + u.Message.Chat.Username
			}

			t.mu.Lock()
			isNew := t.chatID != u.Message.Chat.ID
			t.chatID = u.Message.Chat.ID
			t.chatName = name
			t.mu.Unlock()

			if isNew {
				saveChat(u.Message.Chat.ID, name)
				log.Printf("telegram: connected to chat %d (%s)", u.Message.Chat.ID, name)
			}
			if isNew || strings.HasPrefix(u.Message.Text, "/start") {
				_ = t.SendMessage(ctx, "✅ Connected to Thai Bus Watch!\n\nBus alerts will arrive in this chat. Open the web app to start tracking a bus.")
			}
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
