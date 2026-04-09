SHELL := /bin/zsh

ROOT ?=

.PHONY: dev serve frontend-dev frontend-build build test test-web

dev:
	@set -euo pipefail; \
	npm --prefix web run dev & \
	frontend_pid=$$!; \
	trap 'kill $$frontend_pid 2>/dev/null || true' EXIT INT TERM; \
	if [[ -n "$(ROOT)" ]]; then \
		RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747 --root "$(ROOT)"; \
	else \
		RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747; \
	fi

serve:
	@if [[ -n "$(ROOT)" ]]; then \
		go run ./cmd/relay serve --port 4747 --root "$(ROOT)"; \
	else \
		go run ./cmd/relay serve --port 4747; \
	fi

frontend-dev:
	npm --prefix web run dev

frontend-build:
	npm --prefix web run build
	rm -rf internal/frontend/embed/*
	cp -R web/out/. internal/frontend/embed/
	touch internal/frontend/embed/.gitkeep

build: frontend-build
	mkdir -p bin
	go build -ldflags "-X main.version=dev -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) -X main.date=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/relay ./cmd/relay

test:
	go test ./...

test-web:
	npm --prefix web test
