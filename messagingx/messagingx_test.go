package messagingx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockTelegram serves canned getUpdates and captures sendMessage payloads.
func mockTelegram(t *testing.T, sent *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			io.WriteString(w, `{"ok":true,"result":[
				{"update_id":10,"message":{"chat":{"id":-100123},"from":{"id":555},"text":"/total"}},
				{"update_id":11,"edited_message":{"text":"ignored"}}
			]}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, sent)
			io.WriteString(w, `{"ok":true,"result":{}}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

func TestTelegram_Poll(t *testing.T) {
	srv := mockTelegram(t, &map[string]string{})
	defer srv.Close()
	tg := NewTelegram("TOKEN", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	if tg.Channel() != "telegram" {
		t.Fatalf("channel = %s", tg.Channel())
	}
	updates, next, err := tg.Poll(context.Background(), 0)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("updates = %d, want 1 (edited_message ignored)", len(updates))
	}
	u := updates[0]
	if u.ChatID != "-100123" || u.SenderID != "555" || u.Text != "/total" {
		t.Errorf("update = %+v, want chat -100123 sender 555 text /total", u)
	}
	if next != 12 {
		t.Errorf("nextOffset = %d, want 12 (max update_id + 1)", next)
	}
}

func TestTelegram_Send(t *testing.T) {
	sent := map[string]string{}
	srv := mockTelegram(t, &sent)
	defer srv.Close()
	tg := NewTelegram("TOKEN", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	if err := tg.Send(context.Background(), "-100123", "halo keluarga"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if sent["chat_id"] != "-100123" || sent["text"] != "halo keluarga" {
		t.Errorf("sent payload = %+v, want chat_id -100123 text 'halo keluarga'", sent)
	}
}

func TestTelegram_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"description":"Unauthorized"}`)
	}))
	defer srv.Close()
	tg := NewTelegram("BAD", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err := tg.Send(context.Background(), "1", "x"); err == nil {
		t.Error("expected error for ok:false response")
	}
}

// Provider is implemented by *Telegram.
var _ Provider = (*Telegram)(nil)
