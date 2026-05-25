//go:build tools

// Package tools pins dev-tool dependencies so they appear in go.mod / go.sum
// and can be installed with `go install`. These are never compiled into
// production binaries because of the build tag.
package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
