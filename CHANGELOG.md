# Changelog

All notable changes to **OpenBooklet** will be documented in this file.

The format is based on **[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/)**,
and this project adheres to **[Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html)**.

Release notes are generated from these entries + conventional commit history.
Contributors add entries to the `[Unreleased]` section in the same PR that
delivers the change (see [CONTRIBUTING.md § CHANGELOG](CONTRIBUTING.md#changelog--keep-a-changelog)).

Types of changes:

- **Added** for new features.
- **Changed** for changes in existing functionality.
- **Deprecated** for soon-to-be-removed features.
- **Removed** for now-removed features.
- **Fixed** for any bug fixes.
- **Security** in case of vulnerabilities.

---

## [Unreleased]

**Phase 0 — Specification work-in-progress. No tagged releases yet.**

### Added

- Complete 75-section product specification: [PLAN.md](PLAN.md).
  Covers identity, core abstractions, UI, backend, providers, format,
  phases 0–16, MVP scope, testing strategy, and the 10 design principles.
- Senior Engineer handbook: [AGENTS.md](AGENTS.md).
  Covers persona, domain expertise, architecture, Go/TS conventions,
  testing strategy (including 5 non-negotiable critical tests),
  conventional commits, phase discipline, failure-mode playbook, and key patterns.
- Technical architecture document: [ARCHITECTURE.md](ARCHITECTURE.md).
  Package layout, interface contracts, data-model diagrams, storage design,
  API reference, streaming protocol, non-functional requirements, and ADR index.
- Project landing & documentation: [README.md](README.md).
  Identity, why OpenBooklet, abstractions, MVP golden path, 10 principles,
  tech stack, roadmap snapshot, build instructions, `.obk` example, security notes.
- Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md).
  Prerequisites, phase discipline, branching, conventional commits,
  coding standards, testing, CHANGELOG process, DCO sign-off, PR process.
- Security policy: [SECURITY.md](SECURITY.md).
  Supported versions, private disclosure path (GitHub Security Advisory),
  90-day window, scope (incl. prompt-injection classification), secret-handling guide.
- Community Code of Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
  Contributor Covenant 2.1 with enforcement ladder and private reporting channel.
- Apache 2.0 license: [LICENSE](LICENSE).
- GitHub templates:
  - Bug report (`.github/ISSUE_TEMPLATE/bug_report.md`) — with OpenBooklet-specific
    fields: provider, affected module, `.obk` upload guidance, severity.
  - Feature request (`.github/ISSUE_TEMPLATE/feature_request.md`) — with
    PLAN.md phase mapping, MVP alignment.
  - Pull request (`.github/PULL_REQUEST_TEMPLATE.md`) — conventional commit
    title checklist, AGENTS.md/PLAN.md alignment, round-trip test block,
    no-vendor-leak sanity check, DCO sign-off.
  - Funding (`.github/FUNDING.yml`).

### Changed

- _(none yet)_

### Deprecated

- _(none yet)_

### Removed

- _(none yet)_

### Fixed

- _(none yet)_

### Security

- _(none yet)_

---

## [0.1.0] — YYYY-MM-DD

> **Planned.** Phase 6 (AI Booklet Creation) milestone. First end-to-end demo.
> Everything below is a placeholder. Update before cutting the tag.

### Added

- Go backend skeleton — Phase 1 through Phase 5.
  - `internal/booklet`, `internal/section` models + services.
  - `internal/storage` with SQLite + filesystem repositories.
  - `internal/booklet/serializer.go` — `.obk` v1 round-trip.
  - `internal/booklet/parser.go` — Markdown ⇄ Cells lossless conversion.
  - `internal/provider` interface + `providers/ollama` + `providers/compatible`.
  - `internal/llm` streaming orchestration.
- Initial REST API (`/api/v1`) per [ARCHITECTURE.md § API](ARCHITECTURE.md).
- CLI (`cmd/openbooklet`): `init`, `create`, `open`, `import`, `export`, `generate`, `serve`, `config`, `providers`.
- Web UI (React 18 + TS strict + Vite) — Phase 7 skeleton:
  - Booklet outline sidebar.
  - Chat-based booklet creation screen.
  - Cell editor (prompt + markdown + preview + AI controls).
  - Provider selector + settings.
- E2E Playwright test for the MVP golden path.
- CI via GitHub Actions: lint, vet, test, build on every PR.
- Dockerfile + docker-compose.yml for local evaluation.

### Fixed

- _(populate per actual release)_

### Security

- Context builder instruction/data separation (PLAN.md §42) with regression tests.
- `config.Secret` wrapper prevents accidental key/token logging.

---

## [1.0.0] — YYYY-MM-DD

> **Planned.** Phase 16 complete. Full end-to-end shippable product.
> Release notes will be generated from the CHANGELOG entries of every 0.x release.

### Added

- Everything required by [README.md MVP Golden Path](README.md#mvp---golden-path-v01) plus
  Phases 8–16: Context Engine, Templates, File Intelligence, Review Engine,
  Export (Markdown, HTML, PDF, DOCX), Git Integration, Diagrams, Advanced AI Workflows.

---

## Versioning Policy

- **MAJOR** — incompatible API changes to:
  - `/api/vN/` endpoints or their request/response schemas.
  - The `Provider`, `Exporter`, `Repository` Go interfaces.
  - The `.obk` top-level schema (breaking changes to v1 require `.obk` v2 + migration path).
- **MINOR** — backwards-compatible feature additions per phase.
- **PATCH** — backwards-compatible bug and security fixes only.

During the pre-1.0 phase (0.x.y):

- Minor bumps may carry breaking changes; we will clearly flag them in
  `BREAKING CHANGE:` conventional-commit footers and in the corresponding
  CHANGELOG entry. Pin the exact version if you integrate 0.x in a pipeline.

---

## How to Contribute to This File

See the **CHANGELOG — Keep a Changelog** section in
[CONTRIBUTING.md](CONTRIBUTING.md#changelog--keep-a-changelog). The short version:
every non-trivial PR adds an entry to `[Unreleased]` under the correct heading.
Maintainers promote the `[Unreleased]` block to a version on release day and
create a new empty `[Unreleased]`.
