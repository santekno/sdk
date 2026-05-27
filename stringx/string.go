// Package stringx provides string helper utilities.
package stringx

import (
	"strings"
	"unicode"
)

// IsEmpty reports whether s is empty or contains only whitespace.
func IsEmpty(s string) bool { return strings.TrimSpace(s) == "" }

// Truncate trims s to at most n characters. If truncated, "..." is appended.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// WordCount returns the number of whitespace-separated words in s.
func WordCount(s string) int { return len(strings.Fields(s)) }

// Capitalize returns s with the first letter upper-cased.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// PadLeft pads s on the left with pad up to length n.
func PadLeft(s, pad string, n int) string {
	for len(s) < n {
		s = pad + s
	}
	return s
}

// PadRight pads s on the right with pad up to length n.
func PadRight(s, pad string, n int) string {
	for len(s) < n {
		s += pad
	}
	return s
}

// ToCamel converts a snake_case or kebab-case string to camelCase.
func ToCamel(s string) string {
	parts := splitOnSeparators(s)
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(parts[0]))
	for _, p := range parts[1:] {
		b.WriteString(Capitalize(strings.ToLower(p)))
	}
	return b.String()
}

// ToSnake converts a camelCase or kebab-case string to snake_case.
func ToSnake(s string) string { return joinLowered(splitOnSeparators(s), "_") }

// ToKebab converts a camelCase or snake_case string to kebab-case.
func ToKebab(s string) string { return joinLowered(splitOnSeparators(s), "-") }

// Slugify converts s to a URL-safe slug (lowercase, hyphen-separated).
func Slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func splitOnSeparators(s string) []string {
	var parts []string
	var current strings.Builder
	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && current.Len() > 0 {
			last := []rune(current.String())[current.Len()-1]
			if unicode.IsLower(last) {
				parts = append(parts, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func joinLowered(parts []string, sep string) string {
	lowered := make([]string, len(parts))
	for i, p := range parts {
		lowered[i] = strings.ToLower(p)
	}
	return strings.Join(lowered, sep)
}
