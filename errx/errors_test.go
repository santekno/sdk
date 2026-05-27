package errx_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/santekno/sdk/errx"
)

func TestNew(t *testing.T) {
	err := errx.New("TEST_CODE", "test message")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Code != "TEST_CODE" {
		t.Errorf("Code = %q, want %q", err.Code, "TEST_CODE")
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if len(err.Stack) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestNewf(t *testing.T) {
	err := errx.Newf("FMT_CODE", "value is %d", 42)
	if err.Message != "value is 42" {
		t.Errorf("Message = %q, want %q", err.Message, "value is 42")
	}
}

func TestError(t *testing.T) {
	err := errx.New("MY_CODE", "my message")
	s := err.Error()
	if !strings.Contains(s, "MY_CODE") {
		t.Errorf("Error() = %q, want to contain MY_CODE", s)
	}
	if !strings.Contains(s, "my message") {
		t.Errorf("Error() = %q, want to contain 'my message'", s)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	wrapped := errx.Wrap(cause, "WRAP_CODE", "wrapped")

	if wrapped.Code != "WRAP_CODE" {
		t.Errorf("Code = %q, want %q", wrapped.Code, "WRAP_CODE")
	}
	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is should find the original cause")
	}
	if wrapped.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("root")
	wrapped := errx.Wrapf(cause, "CODE", "msg %s", "extra")
	if wrapped.Message != "msg extra" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "msg extra")
	}
}

func TestWithHTTPStatus(t *testing.T) {
	err := errx.New("NOT_FOUND", "not found").WithHTTPStatus(http.StatusNotFound)
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusNotFound)
	}
}

func TestWithDetail(t *testing.T) {
	err := errx.New("CODE", "msg").WithDetail("user_id", "u-123")
	if err.Details["user_id"] != "u-123" {
		t.Errorf("Details[user_id] = %v, want u-123", err.Details["user_id"])
	}
}

func TestWithDetails(t *testing.T) {
	err := errx.New("CODE", "msg").WithDetails(map[string]any{
		"a": 1,
		"b": "two",
	})
	if err.Details["a"] != 1 || err.Details["b"] != "two" {
		t.Errorf("WithDetails did not merge correctly: %v", err.Details)
	}
}

func TestCode(t *testing.T) {
	err := errx.New("MY_CODE", "msg")
	if got := errx.Code(err); got != "MY_CODE" {
		t.Errorf("Code() = %q, want %q", got, "MY_CODE")
	}
	if got := errx.Code(errors.New("plain")); got != "" {
		t.Errorf("Code() on plain error = %q, want empty", got)
	}
}

func TestHTTPStatusAccessor(t *testing.T) {
	err := errx.New("E", "m").WithHTTPStatus(422)
	if got := errx.HTTPStatus(err); got != 422 {
		t.Errorf("HTTPStatus() = %d, want 422", got)
	}
	if got := errx.HTTPStatus(errors.New("plain")); got != 0 {
		t.Errorf("HTTPStatus() on plain error = %d, want 0", got)
	}
}

func TestDetailsAccessor(t *testing.T) {
	err := errx.New("E", "m").WithDetail("k", "v")
	d := errx.Details(err)
	if d["k"] != "v" {
		t.Errorf("Details() = %v, want k=v", d)
	}
}

func TestWrappedChain(t *testing.T) {
	sentinel := errx.New("SENTINEL", "sentinel error")
	wrapped := errx.Wrap(sentinel, "OUTER", "outer message")
	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is should find sentinel through chain")
	}
	if errx.Code(wrapped) != "OUTER" {
		t.Errorf("Code of outer = %q, want OUTER", errx.Code(wrapped))
	}
}

func ExampleNew() {
	err := errx.New("USER_NOT_FOUND", "user does not exist").
		WithHTTPStatus(404).
		WithDetail("user_id", "u-42")
	_ = err
}
