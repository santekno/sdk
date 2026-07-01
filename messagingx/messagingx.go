// Package messagingx provides a channel-agnostic chat messaging provider and a
// Telegram implementation (Bot API). It lets an application receive group
// messages (long-poll) and send replies without binding to a specific channel;
// a WhatsApp provider can implement the same Provider interface later.
//
//	tg := messagingx.NewTelegram(token)
//	updates, next, _ := tg.Poll(ctx, 0)
//	_ = tg.Send(ctx, updates[0].ChatID, "halo")
package messagingx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Update is one inbound message, normalized across channels.
type Update struct {
	ID       int64  // provider update id (for offset/ack)
	ChatID   string // group/chat id
	SenderID string // sender account id
	Text     string
}

// Provider is a chat channel that can receive and send text messages.
type Provider interface {
	// Channel returns the channel name, e.g. "telegram".
	Channel() string
	// Poll fetches updates after `offset` and returns them plus the next offset.
	Poll(ctx context.Context, offset int64) (updates []Update, nextOffset int64, err error)
	// Send delivers a text message to a chat.
	Send(ctx context.Context, chatID, text string) error
}

// Telegram is a Provider backed by the Telegram Bot API.
type Telegram struct {
	token   string
	baseURL string
	hc      *http.Client
}

// Option configures a Telegram provider.
type Option func(*Telegram)

// WithBaseURL overrides the Bot API base URL (used in tests).
func WithBaseURL(u string) Option { return func(t *Telegram) { t.baseURL = u } }

// WithHTTPClient sets the HTTP client.
func WithHTTPClient(hc *http.Client) Option { return func(t *Telegram) { t.hc = hc } }

// NewTelegram builds a Telegram provider from a bot token.
func NewTelegram(token string, opts ...Option) *Telegram {
	t := &Telegram{
		token:   token,
		baseURL: "https://api.telegram.org",
		hc:      &http.Client{Timeout: 65 * time.Second},
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Channel implements Provider.
func (t *Telegram) Channel() string { return "telegram" }

func (t *Telegram) method(name string) string {
	return fmt.Sprintf("%s/bot%s/%s", t.baseURL, t.token, name)
}

type tgResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Text string `json:"text"`
	} `json:"message"`
}

// Poll implements Provider using getUpdates long-polling.
func (t *Telegram) Poll(ctx context.Context, offset int64) ([]Update, int64, error) {
	url := fmt.Sprintf("%s?offset=%d&timeout=0", t.method("getUpdates"), offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, offset, err
	}
	var resp tgResponse
	if err := t.do(req, &resp); err != nil {
		return nil, offset, err
	}
	var raw []tgUpdate
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, offset, err
	}

	next := offset
	var out []Update
	for _, u := range raw {
		if u.UpdateID >= next {
			next = u.UpdateID + 1
		}
		if u.Message == nil {
			continue
		}
		out = append(out, Update{
			ID:       u.UpdateID,
			ChatID:   strconv.FormatInt(u.Message.Chat.ID, 10),
			SenderID: strconv.FormatInt(u.Message.From.ID, 10),
			Text:     u.Message.Text,
		})
	}
	return out, next, nil
}

// Send implements Provider using sendMessage.
func (t *Telegram) Send(ctx context.Context, chatID, text string) error {
	body, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.method("sendMessage"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	var resp tgResponse
	return t.do(req, &resp)
}

func (t *Telegram) do(req *http.Request, out *tgResponse) error {
	res, err := t.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("messagingx: decode telegram response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("messagingx: telegram error: %s", out.Description)
	}
	return nil
}
