package testx

import (
	"fmt"
	"strings"
	"testing"
)

// AssertNoError fails the test if err is non-nil.
func AssertNoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v%s", err, formatMsg(msgAndArgs))
	}
}

// AssertError fails the test if err is nil.
func AssertError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		t.Errorf("expected an error but got nil%s", formatMsg(msgAndArgs))
	}
}

// AssertEqual fails the test if got != want (using fmt.Sprintf comparison).
func AssertEqual[T comparable](t testing.TB, got, want T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v%s", got, want, formatMsg(msgAndArgs))
	}
}

// AssertNotEqual fails the test if got == want.
func AssertNotEqual[T comparable](t testing.TB, got, unexpected T, msgAndArgs ...any) {
	t.Helper()
	if got == unexpected {
		t.Errorf("expected value to differ from %v%s", got, formatMsg(msgAndArgs))
	}
}

// AssertContains fails the test if haystack does not contain needle.
func AssertContains(t testing.TB, haystack, needle string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%q does not contain %q%s", haystack, needle, formatMsg(msgAndArgs))
	}
}

// AssertTrue fails the test if condition is false.
func AssertTrue(t testing.TB, condition bool, msgAndArgs ...any) {
	t.Helper()
	if !condition {
		t.Errorf("expected true but got false%s", formatMsg(msgAndArgs))
	}
}

// AssertFalse fails the test if condition is true.
func AssertFalse(t testing.TB, condition bool, msgAndArgs ...any) {
	t.Helper()
	if condition {
		t.Errorf("expected false but got true%s", formatMsg(msgAndArgs))
	}
}

// AssertNil fails the test if v is not nil.
func AssertNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if v != nil {
		t.Errorf("expected nil but got %v%s", v, formatMsg(msgAndArgs))
	}
}

func formatMsg(args []any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return fmt.Sprintf(": %v", args[0])
	}
	return ": " + fmt.Sprintf(args[0].(string), args[1:]...)
}
