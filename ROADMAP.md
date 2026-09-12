# OpenBooklet Roadmap

Phase checklist. Checked phases are implemented, tested, and committed;
unchecked phases are upcoming work defined in [PLAN.md](PLAN.md).

## Done

- [x] **Phase 0 — Specification** — PLAN.md, ARCHITECTURE.md, community docs
- [x] **Phase 1 — Core Go Engine** — Booklet/Section models, services, in-memory repos, status state machines
- [x] **Phase 2 — Storage** — GORM ORM (SQLite default; Postgres/MySQL via `OPENBOOKLET_DB_DRIVER`), versioned migrations with NULL backfills, file-backed + upgrade tests
- [x] **Phase 3 — `.obk` Format** — Versioned YAML serializer/deserializer with header/footer fields, golden-file + lossless round-trip tests
- [x] **Phase 4 — Markdown Parser** — Fence-aware `md ⇄ sections`, front matter + preamble preserved, export/import endpoints
- [x] **Phase 5 — LLM Engine** — Provider interface (+ `Name`), shared contract suite, OpenAI-compatible (SSE) + Ollama (NDJSON) clients with retries, timeouts, cancellation
- [x] **Phase 6 — AI Booklet Creation** — `internal/llm` event streaming, booklet generation + per-section regenerate/expand/shorten/edit with history snapshots, all inside the web UI
- [x] **Phase 7 — Web UI (working slice)** — React + TS strict + Vite + Zustand + `marked`
  - [x] Sidebar with booklet list, collapsible icon rail
  - [x] Numbered sections (`1. ## Title`) with Edit/Preview tabs, inline title editing, accordion collapse
  - [x] Manual add-section form (H1–H6, auto-nesting) and section delete with confirmation
  - [x] Chat-style Generate panel with live SSE streaming
  - [x] Per-section AI actions (Regenerate, Expand, Shorten, AI edit)
  - [x] Full-booklet Preview tab with Markdown header/footer blocks
  - [x] Markdown download + file import
  - [x] Booklet rename, metadata editing, delete with confirmation

## Planned

- [ ] **Phase 8 — Context Engine** — `@section` references, file context, token estimation, context preview
- [ ] **Phase 9 — Templates** — Built-in SOP/MOP/runbook/article templates + custom YAML templates
- [ ] **Phase 10 — File Intelligence** — `.md .txt .yaml .json .xml .csv .log .conf` context parsers
- [ ] **Phase 11 — Review Engine** — Completeness/consistency/terminology checks as findings, never silent edits
- [ ] **Phase 12 — Export** — HTML (then PDF/DOCX) alongside Markdown
- [ ] **Phase 13 — Git Integration** — Repo detection, status, diff, commit flows
- [ ] **Phase 14 — Diagrams** — Mermaid rendering in previews (done, lazy-loaded); diagram generation planned
- [ ] **Phase 15 — Advanced AI Workflows** — Whole-booklet actions, batch generation, dependent regen, doc conversion
- [ ] **Phase 16 — v1.0** — Full end-to-end shippable product (releases, Docker, CI)
