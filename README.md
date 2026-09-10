# OpenBooklet

> **Chat is the creation interface. Cells are the editing interface. Markdown is the content layer. The Booklet is the document. AI is the assistant.**

OpenBooklet is an open-source, AI-native documentation workspace for creating, editing, reviewing, organizing, and exporting structured documents — SOPs, MOPs, runbooks, ADRs, architecture documents, technical docs, and internal knowledge.

You start by describing what you want in a chat-style prompt. The AI generates an initial Markdown document. OpenBooklet then **automatically analyzes the Markdown headings and turns them into an editable hierarchy of cells**. Each cell owns its own prompt, Markdown content, context, dependencies, and version history. The resulting Booklet is a versionable, reviewable, Git-friendly documentation artifact stored in a human-readable `.obk` YAML file.

---

## Status

> **Phase 0 — Specification / pre-MVP.** The project is currently in detailed design. See [PLAN.md](PLAN.md) for the full product specification (75 sections) and [ARCHITECTURE.md](ARCHITECTURE.md) for the technical design. The codebase is being built toward the v0.1 golden path (see [MVP](#mvp---golden-path-v01) below).

---

## Why OpenBooklet?

A traditional AI chat gives you a one-shot response and a conversation history. OpenBooklet gives you a **document**.

| Traditional AI Chat | OpenBooklet |
|---------------------|-------------|
| Response is a transient message | Response becomes editable document content |
| Output lives in chat history | Output lives in a structured section tree |
| No built-in way to edit sub-sections | Every heading is its own cell with prompt + editor |
| No per-section AI actions | Regenerate, expand, shorten, review per cell |
| No explicit context management | Reference `@section`, `@booklet`, `@file` as context |
| No document status lifecycle | Draft → Generated → Edited → Reviewed → Approved |
| No per-section version history | Every cell has diff/restore/compare |
| No portable source-of-truth format | `.obk` YAML — Git-friendly, round-trip lossless |

---

## Core Abstractions

```
BOOKLET = DOCUMENT
SECTION = CELL
CELL    = PROMPT + EDITABLE MARKDOWN + CONTEXT + HISTORY
AI      = ASSISTANT
USER    = AUTHOR
.obk    = SOURCE OF TRUTH
```

- **Booklet** — the complete document: metadata, instructions, context, references, settings, and a tree of sections.
- **Section / Cell** — one document section with its own Prompt, Markdown content, Context references, Dependencies, Status, Generation metadata, and Version History.
- **Context Engine** — builds the ordered context window sent to the LLM: System → Application → Booklet Instructions → Section Instructions → Referenced sections → Attached files → Prompt. External files are *data*, never instructions (prompt-injection safe).
- **Provider Abstraction** — pluggable `Provider` interface with `Chat`, `Stream`, `Models`. No vendor lock-in; swap OpenAI, Anthropic, Gemini, Ollama, OpenAI-compatible, Azure, OpenRouter, vLLM, LM Studio, LocalAI.
- **`.obk` Format** — human-readable YAML, Git-friendly, round-trip lossless. The source of truth for every booklet.

---

## MVP — Golden Path (v0.1)

1. Create a booklet → type a chat-style description.
2. AI streams Markdown into the editor.
3. Markdown parser walks headings and auto-creates a hierarchy of cells.
4. Each cell exposes a Prompt editor + Markdown editor + Preview + AI controls.
5. Regenerate / Expand / Shorten / AI-edit individual cells.
6. Reference other sections (`@section:id`) or attach files as context.
7. Save → `.obk` file + SQLite updated.
8. Commit `.obk` to Git.
9. Export to Markdown or HTML.

Everything in the MVP serves this chain. Everything else is deferred until the chain is solid.

---

## The 10 Design Principles

1. **Local-first** — works offline; no cloud dependency required.
2. **AI-optional** — a great Markdown+cells editor even without any LLM configured.
3. **Provider-agnostic** — the application never imports a vendor SDK directly.
4. **Human-controlled** — AI never silently modifies or publishes; every change is visible/revertible.
5. **Markdown-native** — Markdown → Cells → Markdown must be lossless.
6. **Git-friendly** — `.obk` produces readable, meaningful diffs.
7. **Structured documents** — documents are sections with hierarchy, not chat transcripts.
8. **Extensible** — templates, providers, exporters, integrations via interfaces.
9. **Open-source** — avoid proprietary lock-in; prefer protocols over SDKs.
10. **Self-hostable** — deployable on a laptop or private infra with zero external services.

---

## Built-In Document Templates

Article · SOP · MOP · Guideline · Runbook · Technical Documentation · Architecture Document · Custom YAML-defined templates.

---

## Planned Document Operations

Generate · Explain · Rewrite · Expand · Shorten · Summarize · Correct · Improve · Structure · Review · Validate · Translate · Compare · Extract · Convert (e.g. Tech Doc → SOP → MOP).

---

## Tech Stack

| Layer | Choice | Notes |
|-------|--------|-------|
| **Backend Language** | Go | Static binary, excellent stdlib, great concurrency for streaming. |
| **Primary Storage** | SQLite + Filesystem | SQLite for indexes/metadata, filesystem for `.obk`, attachments, exports. Portable to Postgres/object storage later. |
| **Frontend** | React 18 + TypeScript (strict) + Vite | No SSR initially; pure single-page app. |
| **Frontend State** | Zustand (domain-split stores) | `bookletStore`, `sectionStore`, `providerStore`, `uiStore`. |
| **Markdown Editing** | TBD (CodeMirror / Monaco / TipTap) | Must support split edit+preview, syntax highlighting, Mermaid, tables, code blocks, task lists, undo/redo. |
| **Streaming Transport** | HTTP SSE (`text/event-stream`) | Events: `start → token* → metadata? → complete \| error`. AbortController for cancel. |
| **API Style** | REST + JSON | Versioned `/api/v1/`. Pagination via cursors where needed. |
| **Identity / Auth** | None for MVP (local-first) | Plug-point for Phase v1.1+. |
| **Container** | Docker + docker-compose | Postgres optional service in later phases. |
| **CI/CD** | GitHub Actions | Lint, vet, test, build, scan, cross-platform release, Docker push. |

Go package layout (see [ARCHITECTURE.md](ARCHITECTURE.md)):

```
cmd/openbooklet/main.go   ← composition root
internal/
  booklet/   section/   context/   llm/   provider/
  storage/   file/      template/  review/ reference/
  export/    git/       config/    security/
providers/   ← openai, anthropic, gemini, ollama, compatible
web/         ← React/TS/Vite
templates/   ← article.yaml, sop.yaml, mop.yaml, ...
```

---

## Roadmap Snapshot (Phases 0–16)

Full phase definitions are in [PLAN.md §47–§68](PLAN.md).

| Phase | Name | Deliverable |
|-------|------|-------------|
| 0 | Specification | ARCHITECTURE.md, BOOKLET_FORMAT.md, API.md, CONTEXT.md, PROVIDERS.md |
| 1 | Core Go Engine | Booklet/Section models + services + in-memory repos + tests |
| 2 | Storage | SQLite + FS repositories plugged in; same tests pass |
| 3 | `.obk` Format | Serializer/Deserializer; lossless round-trip tests with golden files |
| 4 | Markdown Parser | `md ⇄ sections`, lossless. Milestone — must be solid. |
| 5 | LLM Engine | Provider interface + 2 real providers + streaming + cancel |
| 6 | AI Booklet Creation | Chat prompt → AI → Markdown → Parser → Cells. **First demo milestone.** |
| 7 | Web UI | Sidebar + cell editor + prompt editor + preview + AI controls |
| 8 | Context Engine | Cross-section refs, file context, token estimation, preview |
| 9 | Templates | 7 built-in + custom YAML templates |
| 10 | File Intelligence | `.md .txt .yaml .json .xml .csv .log .conf` parsers |
| 11 | Review Engine | Completeness/consistency/terminology/clarity checks; findings not silent edits |
| 12 | Export | Markdown + HTML (MVP), then PDF/DOCX |
| 13 | Git Integration | Repo detection, status, diff, commit; `.obk` diff readability |
| 14 | Diagrams | Mermaid rendering, then diagram generation |
| 15 | Advanced AI Workflows | Whole-booklet actions, batch gen, dependent regen, doc conversion |
| 16 | **v1.0** | Full end-to-end shippable product |

---

## Getting Started (Once v0.1 ships)

> **Developer previews only during Phases 1–5.** End-user install instructions will be added here once Phase 7 lands.

### From a Release

Download the binary for your platform (Linux/amd64, Linux/arm64, macOS/amd64, macOS/arm64, Windows/amd64) from the Releases page, or pull the Docker image:

```bash
# Docker preview (future)
docker run -p 3000:3000 -v ./data:/data ghcr.io/openbooklet/openbooklet:latest openbooklet serve
```

### From Source (Developers)

See [CONTRIBUTING.md § Prerequisites](CONTRIBUTING.md#prerequisites) for the exact toolchain.

```bash
# 1. Clone
git clone https://github.com/your-org/openbooklet.git
cd openbooklet

# 2. Backend
make tools          # install go tooling (gofmt, staticcheck, golangci-lint)
make test           # run Go test suite
make build          # produces ./bin/openbooklet

# 3. Frontend
cd web && npm install && npm run build

# 4. Run locally
./bin/openbooklet serve --dev    # --dev = plain HTTP on localhost
```

### Configure an LLM Provider

Ollama local-first example (no API keys required):

```yaml
# ~/.config/openbooklet/config.yaml
provider:
  name: ollama
  endpoint: http://localhost:11434
  model: llama3
```

OpenAI-compatible endpoint (works with OpenRouter, Together, vLLM, LM Studio, Azure with adapter, etc.):

```yaml
provider:
  name: compatible
  endpoint: https://api.openrouter.ai/api/v1
  model: anthropic/claude-sonnet
  api_key: ${OPENROUTER_API_KEY}   # never commit keys
```

---

## `.obk` File Format — Quick Example

```yaml
version: "1"

booklet:
  id: kubernetes-sop
  title: Kubernetes Deployment SOP
  type: sop
  version: "1.0"
  status: draft
  audience: l1-operations

instructions: |
  Use formal technical language.
  Do not invent infrastructure information.

sections:
  - id: purpose
    title: Purpose
    level: 2
    prompt: |
      Explain the purpose of this SOP.
    content: |
      This SOP describes the standard
      Kubernetes deployment procedure.

  - id: prerequisites
    title: Prerequisites
    level: 2
    prompt: |
      Generate the required prerequisites.
    content: |
      - Kubernetes cluster access
      - kubectl
      - Required permissions
    depends_on:
      - purpose
```

Full schema: see [PLAN.md §22 `.obk` File Format](PLAN.md) and (forthcoming) `docs/BOOKLET_FORMAT.md`.

---

## Testing Philosophy

- **Go:** `go test ./... -race -count=1`. Unit → Integration (SQLite :memory:) → httptest API tests → provider contract test suite.
- **Frontend:** Vitest (unit/component) + Playwright (E2E golden path).
- **Non-negotiable round-trip tests:**
  - Markdown ⇄ Cells lossless
  - `.obk` ⇄ Model lossless
  - Context builder prompt-injection resistant
  - Streaming cancel (goleak-checked)
  - Status transition state machine

See [AGENTS.md §5 Testing Strategy](AGENTS.md) and [CONTRIBUTING.md § Testing](CONTRIBUTING.md).

---

## Contributing

We welcome code, docs, templates, bug reports, and design discussions. Please read:

1. [CONTRIBUTING.md](CONTRIBUTING.md) — setup, workflow, PR process, conventional commits, DCO sign-off.
2. [AGENTS.md](AGENTS.md) — the full Senior Engineer guide: architecture, patterns, testing, failure playbook, AI-assisted code rules.
3. [PLAN.md](PLAN.md) — the 75-section product spec your change should align with.
4. [ARCHITECTURE.md](ARCHITECTURE.md) — technical design decisions and package boundaries.

Quick checklist before you open a PR:
- [ ] Branch named `feat/<scope>-<thing>` / `fix/<thing>` / `docs/<thing>` …
- [ ] Conventional commit message
- [ ] `make test` (backend) and `npm run test && npm run lint && npx tsc --noEmit` (frontend) pass
- [ ] Round-trip tests updated if parser/serializer changed
- [ ] Changelog entry added (see [CHANGELOG.md](CHANGELOG.md))
- [ ] Signed-off-by (`git commit -s`) if contributing code

---

## Security

OpenBooklet may process sensitive operational information.

- **Never** commit API keys, passwords, tokens, private keys, or production `.obk` files.
- External files imported as context are treated as untrusted data and physically separated from LLM instructions.
- Sensitive configuration values use a `Secret` wrapper type that never appears in logs, panics, or `String()` output.
- Report security vulnerabilities privately — **do not open a public issue.** See [SECURITY.md](SECURITY.md) for the disclosure process and supported versions.

---

## Project Files Reference

| File | Purpose |
|------|---------|
| [PLAN.md](PLAN.md) | Full 75-section product specification (source of truth for product decisions) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Technical architecture, layering, interface contracts |
| [AGENTS.md](AGENTS.md) | Senior Engineer handbook: coding rules, testing, patterns, failure modes |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute; workflow; commit rules |
| [SECURITY.md](SECURITY.md) | Security policy, supported versions, disclosure |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community standards and enforcement |
| [CHANGELOG.md](CHANGELOG.md) | Release history (Keep a Changelog 1.1) |
| [LICENSE](LICENSE) | Apache License, Version 2.0 |
| `.github/` | Issue/PR templates, funding, CI workflows (coming) |

---

## License

OpenBooklet is distributed under the **Apache License, Version 2.0**. See [LICENSE](LICENSE).

```
Copyright 2026 OpenBooklet Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
```

---

## Community & Support

- **Issues** (bugs, feature ideas) → [GitHub Issues](../../issues) using the templates.
- **Discussions** (design questions, templates, use cases) → [GitHub Discussions](../../discussions).
- **Security reports** → See [SECURITY.md](SECURITY.md) for private disclosure channels.
- **Maintainers / Project leads:**
  - `marcuwynu23` — Creator, Project Lead, Maintainer
  - `iammwwhobuild` — Creator, Project Lead, Maintainer
- **Sponsor the project** → See [`.github/FUNDING.yml`](.github/FUNDING.yml) for current sponsor links (PayPal, GitHub Sponsors, and more).

---

*OpenBooklet is being built by and for the people who get paged at 3 AM. If your runbooks are wrong, the outage is longer. Let's get them right, together.*
