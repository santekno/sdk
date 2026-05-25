package stringx_test

import (
	"testing"

	"github.com/santekno/sdk/stringx"
)

func TestIsEmpty(t *testing.T) {
	if !stringx.IsEmpty("") {
		t.Error("IsEmpty(\"\") should be true")
	}
	if !stringx.IsEmpty("  \t\n") {
		t.Error("IsEmpty whitespace should be true")
	}
	if stringx.IsEmpty("x") {
		t.Error("IsEmpty(x) should be false")
	}
}

func TestTruncate(t *testing.T) {
	if got := stringx.Truncate("hello world", 5); got != "hello..." {
		t.Errorf("Truncate = %q, want hello...", got)
	}
	if got := stringx.Truncate("hi", 5); got != "hi" {
		t.Errorf("Truncate = %q, want hi", got)
	}
	if got := stringx.Truncate("anything", 0); got != "" {
		t.Errorf("Truncate(_, 0) = %q, want empty", got)
	}
}

func TestWordCount(t *testing.T) {
	if n := stringx.WordCount("hello  world\tfoo"); n != 3 {
		t.Errorf("WordCount = %d, want 3", n)
	}
}

func TestCapitalize(t *testing.T) {
	if got := stringx.Capitalize("hello"); got != "Hello" {
		t.Errorf("Capitalize = %q, want Hello", got)
	}
	if got := stringx.Capitalize(""); got != "" {
		t.Errorf("Capitalize empty = %q", got)
	}
}

func TestPad(t *testing.T) {
	if got := stringx.PadLeft("5", "0", 3); got != "005" {
		t.Errorf("PadLeft = %q, want 005", got)
	}
	if got := stringx.PadRight("a", ".", 4); got != "a..." {
		t.Errorf("PadRight = %q, want a...", got)
	}
}

func TestToCamel(t *testing.T) {
	cases := map[string]string{
		"hello_world": "helloWorld",
		"hello-world": "helloWorld",
		"helloWorld":  "helloWorld",
	}
	for in, want := range cases {
		if got := stringx.ToCamel(in); got != want {
			t.Errorf("ToCamel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToSnake(t *testing.T) {
	if got := stringx.ToSnake("helloWorld"); got != "hello_world" {
		t.Errorf("ToSnake = %q", got)
	}
}

func TestToKebab(t *testing.T) {
	if got := stringx.ToKebab("helloWorld"); got != "hello-world" {
		t.Errorf("ToKebab = %q", got)
	}
}

func TestSlugify(t *testing.T) {
	if got := stringx.Slugify("Hello, World!"); got != "hello-world" {
		t.Errorf("Slugify = %q, want hello-world", got)
	}
}
