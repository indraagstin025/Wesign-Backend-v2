# ═══════════════════════════════════════════════════════════
# WeSign Backend — Makefile
# ═══════════════════════════════════════════════════════════

.PHONY: all build run dev test lint clean migrate sqlc docker-up docker-down tidy

APP_NAME   := wesign
GO         := go
BINARY_DIR := bin

# ── Development ──────────────────────────────────────────────

dev: ## Hot reload dengan Air
	air

run: ## Jalankan API server
	$(GO) run ./cmd/api

worker: ## Jalankan background worker
	$(GO) run ./cmd/worker

# ── Build ────────────────────────────────────────────────────

build: ## Build semua binary
	$(GO) build -o $(BINARY_DIR)/api.exe ./cmd/api
	$(GO) build -o $(BINARY_DIR)/worker.exe ./cmd/worker
	# Fase 2: $(GO) build -o $(BINARY_DIR)/migrate.exe ./cmd/migrate  (aktifkan setelah cmd/migrate tersedia)

tidy: ## go mod tidy
	$(GO) mod tidy

# ── Quality ──────────────────────────────────────────────────

test: ## Jalankan semua test
	$(GO) test ./... -v -coverprofile=coverage.out

test-short: ## Jalankan test tanpa integration
	$(GO) test ./... -short -v

lint: ## Jalankan golangci-lint
	golangci-lint run ./...

vet: ## go vet
	$(GO) vet ./...

coverage: ## Test dengan coverage report
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

# ── Database (membutuhkan cmd/migrate — tersedia pada Fase 2) ─

migrate-up: ## Jalankan semua migration
	$(GO) run ./cmd/migrate up

migrate-down: ## Rollback 1 migration terakhir
	$(GO) run ./cmd/migrate down

migrate-status: ## Tampilkan status migration
	$(GO) run ./cmd/migrate status

sqlc: ## Generate code dari sqlc
	sqlc generate

# ── Docker ───────────────────────────────────────────────────

docker-up: ## Start infrastructure dev
	docker compose up -d

docker-down: ## Stop infrastructure dev
	docker compose down

docker-build: ## Build Docker image
	docker build -t $(APP_NAME)-backend:latest .

# ── Utilities ────────────────────────────────────────────────

clean: ## Hapus build artifacts
	rm -rf $(BINARY_DIR) tmp coverage.out coverage.html

seed: ## Seed development data
	echo 'seed belum tersedia — dijadwalkan pada Fase 12 (Test Data Management)'; exit 1

help: ## Tampilkan bantuan
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'