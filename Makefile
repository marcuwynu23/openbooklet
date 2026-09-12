export CGO_ENABLED := 1

# Detect if gcc is available for race detector
GCC_FOUND := $(shell gcc --version 2>/dev/null && echo yes || echo no)
RACE_FLAG := $(if $(filter yes,$(GCC_FOUND)),-race,)

.PHONY: all fmt vet lint test build run clean security go-check frontend-check

all: go-check frontend-check

# --- Go ---

go-check: fmt vet lint test build security

fmt:
	gofmt -d .

fmt-fix:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run

staticcheck:
	staticcheck ./...

test:
	go test ./... $(RACE_FLAG) -count=1

test-race:
	go test ./... -race -count=1

test-covered:
	go test ./... $(RACE_FLAG) -count=1 -coverprofile=coverage.out -covermode=atomic

build:
	go build ./...

run: build
	go run ./cmd/openbooklet

security:
	command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed, skipping"

# --- Frontend ---

frontend-check: tsc lint-frontend prettier-check test-frontend

tsc:
	cd frontend && npx tsc --noEmit

lint-frontend:
	cd frontend && npx eslint src/

prettier-check:
	cd frontend && npx prettier --check src/

prettier-fix:
	cd frontend && npx prettier --write src/

test-frontend:
	cd frontend && npx vitest run

e2e:
	cd frontend && npx playwright test

# --- Utilities ---

clean:
	go clean ./...

help:
	@echo "Available targets:"
	@echo "  all            - Run all quality checks"
	@echo "  go-check       - Go linting, vet, test, build, security"
	@echo "  fmt            - Check Go formatting"
	@echo "  fmt-fix        - Fix Go formatting"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  test           - Run all tests with race detector"
	@echo "  test-covered   - Run tests with coverage"
	@echo "  build          - Build all packages"
	@echo "  run            - Build and run the CLI"
	@echo "  security       - Run govulncheck"
	@echo "  frontend-check - Frontend type-check, lint, format, test"
	@echo "  e2e            - Run Playwright e2e tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help"
