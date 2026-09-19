# WeSign Backend

Platform Tanda Tangan Elektronik Tidak Tersertifikasi — Backend Service.

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Query Layer | sqlc |
| Migration | goose |
| Job Queue | River (PostgreSQL-based) |
| Cache | Redis 7 |
| Object Storage | S3-compatible (MinIO/AWS S3) |
| Observability | Prometheus + OpenTelemetry |

## Arsitektur

Clean Architecture dengan dependency rule:

```
Handler → UseCase → Domain ← Infrastructure (implements interface)
```

Sistem berjalan sebagai **dua proses**:
1. **API Server** (`cmd/api`) — melayani HTTP request
2. **Background Worker** (`cmd/worker`) — menjalankan job async (sealing, email, cleanup)

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- [Air](https://github.com/cosmtrek/air) (hot reload, opsional)
- [sqlc](https://docs.sqlc.dev) (code generation)
- [golangci-lint](https://golangci-lint.run)

### Setup

```bash
# 1. Clone & masuk direktori
git clone <repo-url> && cd WeSign-Backend

# 2. Copy environment config
cp .env.example .env

# 3. Start infrastructure (PostgreSQL, Redis, MinIO, Mailpit)
make docker-up

# 4. Install dependencies
make tidy

# 5. Jalankan migration
make migrate-up

# 6. Generate sqlc code
make sqlc

# 7. Jalankan development server (hot reload)
make dev
```

### Struktur Direktori

```
cmd/          → Entrypoints (api, worker, migrate)
internal/     → Private application code
  auth/       → Modul autentikasi
  document/   → Modul dokumen
  signing/    → Modul signing
  certificate/→ Modul sertifikat
  verification/→ Modul verifikasi publik
  asset/      → Modul aset tanda tangan
  pdp/        → Modul UU PDP
  admin/      → Modul admin
  job/        → Background worker
  shared/     → Shared utilities
  infrastructure/ → External service clients
pkg/          → Public reusable packages
migrations/   → Goose migration files
queries/      → sqlc query files
```

### Available Commands

```bash
make help          # Tampilkan semua command
make dev           # Hot reload development
make build         # Build semua binary
make test          # Jalankan semua test
make lint          # Jalankan linter
make migrate-up    # Jalankan migration
make sqlc          # Generate sqlc code
make docker-up     # Start dev infrastructure
```

## Dokumen

- [URD](docs/URD-WeSign-v2.md) — User Requirements Document
- [PRD](docs/PRD-WeSign-v2.md) — Product Requirements Document
- [SRS](docs/SRS-WeSign-v2.md) — Software Requirements Specification
- [TDD](docs/TDD-Docs/) — Technical Design Document
- [ERD](docs/erd/) — Entity Relationship Diagram

## Lisensi

Proprietary — WeSign