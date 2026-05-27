.PHONY: test lint bench build tidy coverage docs fmt vet security

GO ?= go
GOFLAGS ?=

# Run all tests with race detector
test:
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report written to coverage.out"

# Run tests and enforce per-category coverage floors
coverage: test
	@echo "--- Core packages (≥85%) ---"
	@$(GO) tool cover -func=coverage.out | grep -E "^github.com/santekno/sdk/(slicex|mapx|stringx|numx|uuidx|ptrx|convx|errx|validx|timex)/" | \
		awk '{sum+=$$3; count++} END {pct=sum/count; printf "Coverage: %.1f%%\n", pct; if(pct<85){print "FAIL: core coverage below 85%"; exit 1}}'
	@echo "--- Security packages (≥95%) ---"
	@$(GO) tool cover -func=coverage.out | grep -E "^github.com/santekno/sdk/(cryptox|hashx|apikey|jwtx)/" | \
		awk '{sum+=$$3; count++} END {if(count==0){exit 0}; pct=sum/count; printf "Coverage: %.1f%%\n", pct; if(pct<95){print "FAIL: security coverage below 95%"; exit 1}}'

# Run linter
lint:
	golangci-lint run ./...

# Run benchmarks
bench:
	$(GO) test -bench=. -benchmem -run=^$$ -count=3 ./...

# Build all packages (verify they compile)
build:
	$(GO) build ./...

# Format code
fmt:
	gofmt -w -s .
	goimports -w .

# Vet
vet:
	$(GO) vet ./...

# Tidy go.mod
tidy:
	$(GO) mod tidy

# Security scan
security:
	gosec -severity=high ./...
	govulncheck ./...

# Build docs site
docs:
	cd docs && npm run build
