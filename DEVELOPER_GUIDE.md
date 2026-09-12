# OpenBooklet Developer Guide

## Prerequisites

- Go 1.24+
- Node 22+
- `golangci-lint` (for `make lint`)
- No C compiler needed: SQLite runs via pure-Go GORM drivers.

## Make Targets

| Target | Purpose |
|--------|---------|
| `make go-check` | `gofmt` + `go vet` + `golangci-lint` + `go test` + `go build` (+ `govulncheck` if installed) |
| `make test` | Go tests (race detector when gcc is available) |
| `make run` | Build and serve `http://localhost:8080` |
| `make frontend-install` | `npm install` in `frontend/` |
| `make frontend-build` | `tsc --noEmit` + `vite build` → `frontend/dist/` (required before `make run`) |
| `make frontend-dev` | Vite dev server with `/api` proxy to `:8080` |

Windows note: recipes run under `sh`, so frontend targets call `npm.cmd`/`npx.cmd`.

## Configuration (Environment Variables)

| Variable | Default | Purpose |
|----------|---------|---------|
| `OPENBOOKLET_PORT` | `8080` | HTTP port |
| `OPENBOOKLET_HOST` | `localhost` | Bind address |
| `OPENBOOKLET_DB_DRIVER` | `sqlite` | `sqlite` \| `postgres` \| `mysql` |
| `OPENBOOKLET_DB_PATH` | `./data/openbooklet.db` | File path, `:memory:`, or driver DSN |
| `OPENBOOKLET_STORAGE_PATH` | `./data` | Data directory |
| `OPENBOOKLET_PROVIDER` | _(empty)_ | `ollama` \| `openai` \| `compatible` (empty = AI-optional editor mode) |
| `OPENBOOKLET_MODEL` | `llama3` / `gpt-4o-mini` | Default model per provider |
| `OPENBOOKLET_PROVIDER_ENDPOINT` | provider default | API base URL |
| `OPENBOOKLET_API_KEY` | _(empty)_ | Secret; never logged |
| `OPENBOOKLET_LOG_LEVEL` | `info` | Log level |
| `OPENBOOKLET_TIMEOUT` | `30` | Timeout |

## Testing

```bash
go test ./... -count=1          # backend: unit + SQLite + httptest API + provider contracts
cd frontend && npm run test     # Vitest component tests (accordion, preview, forms)
cd frontend && npm run build    # strict tsc + production bundle
```

Storage changes must extend the file-backed, old-schema-upgrade, NULL-backfill,
and migration-idempotency tests in `internal/storage/sqlite_test.go` — not just
`:memory:` tests. See [AGENTS.md](AGENTS.md) for the full testing strategy.

## Project Layout

```
cmd/openbooklet/      composition root: server, routes (apiServer), provider wiring
internal/
  booklet/            domain: model, service, parser, serializer, generation
  section/            domain: model, service, history
  llm/                generation orchestration (start → token* → complete)
  provider/           Provider interface + shared contract suite
  storage/            GORM models, dialects, versioned migrations, repository
  config/             env-based configuration
providers/            openai (OpenAI-compatible SSE), ollama (NDJSON)
frontend/             React + TS + Vite (src/api.ts, src/stores.ts, src/components/)
```

## HTTP API (`/api/v1/`)

Responses use the `{ data, error?: { code, message } }` envelope.

| Method & Path | Purpose |
|---------------|---------|
| `GET /healthz` | Liveness |
| `GET /api/v1/version` | Version + provider |
| `GET /api/v1/booklets` | List booklets |
| `POST /api/v1/booklets` | Create a booklet |
| `GET /api/v1/booklets/{id}` | Full booklet with sections |
| `PUT /api/v1/booklets/{id}` | Update title/type/audience/instructions/header/footer/showFooter |
| `DELETE /api/v1/booklets/{id}` | Delete (204) |
| `POST /api/v1/booklets/{id}/sections` | Add a section manually |
| `PUT /api/v1/booklets/{id}/sections/{sid}` | Edit title/prompt/content/level |
| `DELETE /api/v1/booklets/{id}/sections/{sid}` | Delete a section (204) |
| `POST /api/v1/booklets/{id}/sections/{sid}/regenerate` | AI rewrite (`regenerate\|expand\|shorten\|edit`) |
| `POST /api/v1/booklets/{id}/generate` | SSE stream: `start → token* → complete \| error` |
| `GET /api/v1/booklets/{id}/export?format=md` | Download canonical Markdown |
| `POST /api/v1/booklets/{id}/import` | Upload a Markdown file as sections (4 MiB cap) |

Conventions, patterns, and the failure-mode playbook live in [AGENTS.md](AGENTS.md).
