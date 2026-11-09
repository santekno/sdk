// Package messagingx provides a channel-agnostic chat messaging provider and a
// Telegram implementation (Bot API). It lets an application receive group
// messages (long-poll) and send replies without binding to a specific channel;
// a WhatsApp provider can implement the same Provider interface later.
//
//	tg := messagingx.NewTelegram(token)
//	updates, next, _ := tg.Poll(ctx, 0)
//	_ = tg.Send(ctx, updates[0].ChatID, "halo")
//
// The Bot API carries the bot token in the request path, so errors returned by
// this package never include the request URL or the token: transport errors
// are rewrapped without their *url.Error, and Telegram's free-text error
// description is reported only when it is a known, data-free message.
// errors.Is with context.Canceled and context.DeadlineExceeded, and errors.As
// with net.Error (Timeout), keep working on the returned errors.
package messagingx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultTimeout bounds one Bot API call made with the default HTTP client.
// Poll uses getUpdates with timeout=0 (no long polling), so a call that takes
// longer than this is stuck rather than waiting for updates.
const defaultTimeout = 15 * time.Second

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

// WithHTTPClient sets the HTTP client. A nil client keeps the default client,
// which has a 15 second timeout. A custom client should set its own Timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(t *Telegram) {
		if hc != nil {
			t.hc = hc
		}
	}
}

// NewTelegram builds a Telegram provider from a bot token. The default HTTP
// client times out each call after 15 seconds; use WithHTTPClient to change it.
func NewTelegram(token string, opts ...Option) *Telegram {
	t := &Telegram{
		token:   token,
		baseURL: "https://api.telegram.org",
		hc:      &http.Client{Timeout: defaultTimeout},
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
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
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
	const name = "getUpdates"
	u := fmt.Sprintf("%s?offset=%d&timeout=0", t.method(name), offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, offset, t.requestErr(name, "build request", err)
	}
	var resp tgResponse
	if err := t.do(name, req, &resp); err != nil {
		return nil, offset, err
	}
	var raw []tgUpdate
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, offset, t.requestErr(name, "decode updates", err)
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
	const name = "sendMessage"
	body, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.method(name), bytes.NewReader(body))
	if err != nil {
		return t.requestErr(name, "build request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	var resp tgResponse
	return t.do(name, req, &resp)
}

func (t *Telegram) do(name string, req *http.Request, out *tgResponse) error {
	res, err := t.hc.Do(req)
	if err != nil {
		return t.requestErr(name, "send request", err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		if res.StatusCode >= http.StatusBadRequest {
			return &apiError{method: name, code: res.StatusCode}
		}
		return t.requestErr(name, "decode response", err)
	}
	if !out.OK {
		e := &apiError{method: name, code: out.ErrorCode, description: safeDescription(out.Description)}
		if e.code == 0 {
			e.code = res.StatusCode
		}
		if out.Parameters != nil && out.Parameters.RetryAfter > 0 {
			e.retryAfter = time.Duration(out.Parameters.RetryAfter) * time.Second
		}
		return e
	}
	return nil
}

// requestErr wraps a request-building, transport, or decoding error so that
// neither the request URL nor the bot token reaches the error text.
func (t *Telegram) requestErr(method, stage string, err error) error {
	return &requestError{method: method, stage: stage, err: redact(err, t.token)}
}

// redact drops the *url.Error layer (its URL holds the token) and, when the
// remaining message still contains the token, replaces the cause with a
// redacted copy that keeps context and timeout classification.
func redact(err error, token string) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		err = ue.Err
	}
	msg := err.Error()
	if token == "" || !containsToken(msg, token) {
		return err
	}
	r := &redactedError{msg: replaceToken(msg, token)}
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) {
		r.timeout = ne.Timeout()
	}
	switch {
	case errors.Is(err, context.Canceled):
		r.cause = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		r.cause = context.DeadlineExceeded
		r.timeout = true
	}
	return r
}

const redactedToken = "<redacted>"

func tokenForms(token string) []string {
	forms := []string{token}
	for _, f := range []string{url.PathEscape(token), url.QueryEscape(token)} {
		if f != token {
			forms = append(forms, f)
		}
	}
	return forms
}

func containsToken(msg, token string) bool {
	for _, f := range tokenForms(token) {
		if strings.Contains(msg, f) {
			return true
		}
	}
	return false
}

func replaceToken(msg, token string) string {
	for _, f := range tokenForms(token) {
		msg = strings.ReplaceAll(msg, f, redactedToken)
	}
	return msg
}

// requestError reports a Bot API call that failed before a usable response
// was decoded. It names the API method, never the URL.
type requestError struct {
	method string
	stage  string
	err    error
}

func (e *requestError) Error() string {
	return "messagingx: telegram " + e.method + ": " + e.stage + ": " + e.err.Error()
}

func (e *requestError) Unwrap() error { return e.err }

// Format prints the redacted message for every verb, including %+v and %#v,
// so struct dumps cannot reach a wrapped value.
func (e *requestError) Format(f fmt.State, verb rune) { formatMessage(f, verb, e.Error()) }

// redactedError stands in for a cause whose text contained the token. It
// keeps context.Canceled/context.DeadlineExceeded for errors.Is and Timeout
// for net.Error checks.
type redactedError struct {
	msg     string
	timeout bool
	cause   error
}

func (e *redactedError) Error() string   { return e.msg }
func (e *redactedError) Unwrap() error   { return e.cause }
func (e *redactedError) Timeout() bool   { return e.timeout }
func (e *redactedError) Temporary() bool { return e.timeout }

func (e *redactedError) Format(f fmt.State, verb rune) { formatMessage(f, verb, e.msg) }

// apiError reports a Bot API response with ok=false or a non-JSON error
// status. Telegram's description is kept only when it is a known message
// without user data; otherwise the HTTP status text is used.
type apiError struct {
	method      string
	code        int
	description string
	retryAfter  time.Duration
}

func (e *apiError) Error() string {
	var b strings.Builder
	b.WriteString("messagingx: telegram ")
	b.WriteString(e.method)
	b.WriteString(": api error ")
	b.WriteString(strconv.Itoa(e.code))
	desc := e.description
	if desc == "" {
		desc = http.StatusText(e.code)
	}
	if desc != "" {
		b.WriteString(" ")
		b.WriteString(desc)
	}
	if e.retryAfter > 0 {
		b.WriteString(" (retry after ")
		b.WriteString(e.retryAfter.String())
		b.WriteString(")")
	}
	return b.String()
}

func (e *apiError) Format(f fmt.State, verb rune) { formatMessage(f, verb, e.Error()) }

// knownDescriptions are Bot API error descriptions that carry no user data.
var knownDescriptions = map[string]struct{}{
	"Unauthorized":                       {},
	"Not Found":                          {},
	"Bad Request: chat not found":        {},
	"Bad Request: message text is empty": {},
	"Bad Request: message is too long":   {},
	"Bad Request: not enough rights to send text messages to the chat":                                             {},
	"Forbidden: bot was blocked by the user":                                                                       {},
	"Forbidden: bot was kicked from the group chat":                                                                {},
	"Forbidden: bot was kicked from the supergroup chat":                                                           {},
	"Forbidden: bot is not a member of the channel chat":                                                           {},
	"Forbidden: bot can't initiate conversation with a user":                                                       {},
	"Forbidden: user is deactivated":                                                                               {},
	"Conflict: terminated by other getUpdates request; make sure that only one bot instance is running":            {},
	"Conflict: can't use getUpdates method while webhook is active; use deleteWebhook to delete the webhook first": {},
}

func safeDescription(d string) string {
	if _, ok := knownDescriptions[d]; ok {
		return d
	}
	return ""
}

func formatMessage(f fmt.State, verb rune, msg string) {
	switch verb {
	case 'q':
		fmt.Fprintf(f, "%q", msg)
	default:
		_, _ = f.Write([]byte(msg))
	}
}
