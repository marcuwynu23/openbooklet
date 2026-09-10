# Contributing to OpenBooklet

First off — thank you for taking the time to contribute. OpenBooklet is being built by and for the people who write the runbooks that get paged at 3 AM. Every improvement, bug fix, template, and piece of feedback makes that experience better for everyone.

This document explains **how** to contribute. Read it together with:

- [PLAN.md](PLAN.md) — the 75-section product spec. Open a discussion if a proposed change deviates from it.
- [ARCHITECTURE.md](ARCHITECTURE.md) — package boundaries, interfaces, technology choices.
- [AGENTS.md](AGENTS.md) — the Senior Engineer guide: coding standards, testing strategy, failure-mode playbook, key patterns.
- [README.md](README.md) — project overview, status, roadmap, and the MVP golden path.

---

## Code of Conduct

All project spaces are governed by the [Contributor Covenant 2.1](CODE_OF_CONDUCT.md). Report violations privately via the channels listed in that document. We do not tolerate harassment, discrimination, or unprofessional behavior of any kind.

---

## What Can I Contribute?

Everything is welcome, **especially** during Phase 0 (Specification). Ideas do not require a PR.

| Contribution Type | How |
|-------------------|-----|
| **Bug reports** | File a [Bug Report](../../issues/new?template=bug_report.md). Include repro steps, environment, and expected/actual behavior. |
| **Feature ideas / Design discussion** | Open a [Feature Request](../../issues/new?template=feature_request.md) or a thread in [Discussions](../../discussions) during Phase 0 — we'd rather debate spec than refactor code. |
| **Documentation** | Fixes to `README.md`, `ARCHITECTURE.md`, doc pages, Go doc comments, TS docstrings. Mark PRs with `docs(scope): ...`. |
| **Templates** | New built-in templates under `templates/`. Must ship with a README entry and a sample generated booklet in `examples/`. |
| **Exporters / Providers** | Implement the `Exporter` or `Provider` interface. Must pass the contract test suite. See [ARCHITECTURE.md § Interfaces](ARCHITECTURE.md). |
| **Code** | See [Good First Issues](../../contribute) and the roadmap in [README.md § Roadmap Snapshot](README.md#roadmap-snapshot-phases-016). |
| **Security issues** | **Do not file a public issue.** Follow [SECURITY.md](SECURITY.md). |

---

## Prerequisites

Make sure the following are installed before you try to build locally:

| Tool | Minimum Version | Notes |
|------|-----------------|-------|
| **Go** | 1.22+ | Required for all backend work. `go env GOPATH` should be on your PATH. |
| **Node.js** | 20 LTS (Iron) | Required for the `web/` frontend. |
| **npm** | 10+ | Shipped with Node 20. Or use `pnpm` / `bun` at your preference (CI uses npm). |
| **SQLite 3** | (shipped with Go driver) | No manual install required — the `mattn/go-sqlite3` or `modernc.org/sqlite` driver bundles it. |
| **golangci-lint** | latest | Installed via `make tools`. Runs in CI; run locally to avoid round trips. |
| **staticcheck** | latest | Installed via `make tools`. |
| **(Optional) Ollama** | any | For end-to-end LLM testing with a local provider. |

If you intend to run Playwright E2E tests:

```bash
cd web && npx playwright install --with-deps chromium
```

---

## Set Up Your Environment

```bash
# 1. Fork the repository on GitHub, then clone your fork.
git clone git@github.com:<your-username>/openbooklet.git
cd openbooklet
git remote add upstream git@github.com:<org>/openbooklet.git

# 2. Install Go tooling
make tools

# 3. Verify the backend builds and tests pass
make test
make build           # produces ./bin/openbooklet

# 4. Set up the frontend
cd web
npm install
npm run typecheck    # TS strict
npm run lint
npm run test
cd ..
```

Then copy the sample config and edit to your taste:

```bash
cp .config.example.yaml ~/.config/openbooklet/config.yaml
```

For development you usually don't need a real LLM provider. The test suite uses in-memory `FakeProvider` exclusively.

---

## Phase Discipline

Read the roadmap before you start coding. Phases build on each other. **Do not jump ahead.**

| Phase | Name | Acceptance bar |
|-------|------|----------------|
| 0 | Specification | Docs, spec, ADRs. No code yet beyond prototypes. |
| 1 | Core Go Engine | Models, services, in-memory repos. 100% of public API unit-tested. No DB calls allowed. |
| 2 | Storage | Plug SQLite + FS repositories. *Same* tests from Phase 1 must still pass. |
| 3 | `.obk` Format | Serializer/deserializer + lossless round-trip via golden files. |
| 4 | Markdown Parser | `md ⇄ sections` lossless. Fuzzed; no panics on malformed input. |
| 5 | LLM Engine | Provider interface + 2 real providers + streaming + cancel + contract tests. |
| 6 | AI Booklet Creation | First end-to-end demo milestone. E2E tests pass. |
| 7–16 | Follow the spec | Each phase must pass its own bar *plus* not regress earlier phases. |

If you are unsure whether a change belongs in the current phase, **open a discussion first.** It is much cheaper to move a phase boundary in a doc than it is to revert a PR.

---

## Branching & Workflow

```
main ← (release tags, production-ready)
  ↑
develop ← (PR target for all non-hotfix work)
  ↑
feature/<scope>-<name>     ← your branch
fix/<scope>-<name>
docs/<scope>-<name>
refactor/<scope>-<name>
test/<scope>-<name>
chore/<scope>-<name>
perf/<scope>-<name>
```

- Fork → branch off `develop`.
- One branch = one logical change. If you describe it with "and", split it.
- Keep rebased on `develop` as PRs ahead of you land.
- Hotfixes (post-release) branch from `main` and target `main`. Maintainers will back-merge into `develop`.

---

## Commit Messages — Conventional Commits 1.0.0

**Every commit message must follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).** This is enforced by CI via commitlint. The CHANGELOG is generated from these messages.

### Format

```
<type>[optional scope][!]: <description>

[optional body — wrap at 72 cols]

[optional footer(s)]
```

### Types

| Type | SemVer Impact | Purpose |
|------|---------------|---------|
| `feat` | **MINOR** | A new user-visible feature. |
| `fix` | **PATCH** | A user-visible bug fix. |
| `perf` | **PATCH** | Performance improvement, no behavior change. |
| `refactor` | None | Code change — no feature, no bug fix. |
| `test` | None | Adding/fixing tests. |
| `docs` | None | Documentation (README, ARCHITECTURE, doc comments that change public-API docs). |
| `style` | None | Whitespace, formatting (prefer `gofmt` / `prettier` run as a pre-commit). Avoid style-only PRs; batch with real changes. |
| `build` | None | Build system, dependencies, Dockerfile, Makefile. |
| `ci` | None | GitHub Actions, CI configs. |
| `chore` | None | Housekeeping: local tool config, housekeeping. |

### Scopes

Use the module/package name when applicable: `booklet`, `section`, `parser`, `serializer`, `llm`, `provider`, `storage`, `context`, `export`, `review`, `template`, `file`, `git`, `api`, `cli`, `ui`, `e2e`, `deps`.

### Breaking Changes

Indicate a breaking change with **both** of the following:

1. A `!` after the type/scope: `feat(api)!: remove inline sections`
2. A footer line starting with exactly `BREAKING CHANGE:` followed by a description + migration path.

This triggers a **MAJOR** SemVer bump.

### Examples

```
feat(parser): support setext-style underlined headings
fix(booklet): preserve section order on reload after reorder UI
test(serializer): add golden file for 8-section SOP booklet with history
refactor(provider): extract shared SSE streaming logic into base client
perf(storage): batch section status updates into single tx
docs(ARCHITECTURE): add section on context builder ordering
chore(deps): bump go-sqlite3 from 1.14.22 to 1.14.23
feat(api)!: replace sections array with cursor pagination

BREAKING CHANGE: GET /api/booklets/{id} no longer returns nested
sections inline. Use GET /api/booklets/{id}/sections?cursor=<last-id>
for paginated results. Migration path documented in
docs/migrations/0001-cursor-pagination.md.
```

### No-Nos

- No WIP commits on merge. Squash them before you request review.
- No empty descriptions.
- No vague messages like `fix bug`, `update`, or (`sigh`) `stuff`.

---

## Coding Standards

See [AGENTS.md §4 Coding Best Practices](AGENTS.md) for the full list. The short version:

### Go

- Run `gofmt -w .` and `golangci-lint run` before committing.
- Wrap errors with context: `fmt.Errorf("opening booklet %q: %w", id, err)`.
- First argument to every IO/network/LLM call is `ctx context.Context`. Respect `ctx.Done()`.
- Exported symbols require a Go doc comment.
- Define interfaces on the consumer side, not the implementation side.
- No `init()` side effects. No global mutable state.
- Tests: table-driven, no randomness without a seed, no real network or real LLM.

### TypeScript / React

- Strict mode on. No `any`. Use `satisfies`, type guards, or `zod` at API boundaries.
- Run `prettier --write src/ && eslint src/` before committing.
- Effects require cleanup when they subscribe (SSE, timers, AbortController).
- Components are pure in props. Derive via `useMemo`, not `useState`+`useEffect`.
- Split stores by domain. Never call `fetch` directly from a component — use `src/services/api.ts`.

### Cross-cutting

- **No vendor SDKs in the app core.** The `internal/` tree must import only `internal/provider/`'s `Provider` interface. Vendor SDKs live in `providers/<name>/` only and behind the interface.
- **No logging of secrets or full documents.** Use the `config.Secret` type and the `security.Redactor` pipeline.
- **IDs are ULID or UUIDv7 strings.** No database auto-increment ints exposed over the API or in `.obk`.

---

## Testing

A passing test suite is **required** on every PR. See [AGENTS.md §5 Testing Strategy](AGENTS.md).

### Run the Go Suite

```bash
make test                     # unit + integration + race detector
go test ./... -race -count=1  # same, explicit
make lint                     # gofmt, vet, staticcheck, golangci-lint
make check                    # lint + test — what CI runs
```

### Run the Frontend Suite

```bash
cd web
npm run test          # Vitest unit & components (watch-mode: npm run test -- --watch)
npm run typecheck     # tsc --noEmit
npm run lint          # eslint
npm run test:e2e      # Playwright E2E (install browsers first: npx playwright install chromium)
```

### The 5 Non-Negotiable Tests

If your change touches any of these areas, the corresponding critical test must pass **and** be updated if the contract changes:

1. **Markdown ⇄ Cells lossless** → add your input/output as a golden file.
2. **`.obk` ⇄ Model lossless** → deep-equal every field; update the golden with `-update`.
3. **Prompt-injection resistance** → the malicious external file is data, not instructions.
4. **Streaming cancel** → `goleak.VerifyTestMain` catches goroutine leaks.
5. **Status transitions** → every edge of the state machine has a table test.

### When Adding Tests

- Table-drive parsers, serializers, validators, state transitions.
- Error-path coverage >= happy-path coverage. For every `if err != nil` you add, add a test.
- Favor one fake (fake repo, fake provider) over layered mocks.
- Prefer golden files + `-update` flag for serialized output.
- Never assert exact timing, exact error messages, or floats.

---

## CHANGELOG — Keep a Changelog

This project uses the **[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/)** format in `CHANGELOG.md`, combined with **Semantic Versioning 2.0**.

On every non-trivial PR, **add an entry to the `[Unreleased]` section** of `CHANGELOG.md` under the appropriate heading:

```markdown
## [Unreleased]

### Added
- New Markdown parser support for setext headings

### Fixed
- Section ordering preserved after UI reorder
```

If you're unsure whether a change warrants a changelog entry, ask yourself: *would a downstream user updating to the next release care about this?* If yes, add it.

---

## DCO — Sign Your Work

OpenBooklet uses the **Developer Certificate of Origin (DCO) v1.1** instead of a CLA. Every commit in a PR must have a `Signed-off-by:` trailer.

> By adding a sign-off you certify that you wrote the change or otherwise have the right to contribute it under the project's open-source license. Full text: https://developercertificate.org/

Sign-off automatically appends the trailer:

```bash
git commit -s -m "feat(parser): support setext headings"
```

Which produces:

```
feat(parser): support setext headings

Signed-off-by: Your Name <your.email@example.com>
```

CI's DCO check fails if any commit is missing the trailer. Fix an unsigned commit with:

```bash
git commit --amend --no-edit -s
git push --force-with-lease
```

---

## Pull Request Process

### Before Opening a PR

- [ ] `develop` is merged or rebased into your branch.
- [ ] `make check` (backend) and `cd web && npm run typecheck && npm run lint && npm run test` (frontend) pass.
- [ ] You have self-reviewed `git diff develop...HEAD` line by line.
- [ ] New code has tests, docs, and a CHANGELOG entry (as applicable).
- [ ] Every commit is conventional and signed off.

### Open the PR

1. Target: **`develop`** (unless it's a hotfix, in which case target `main`).
2. The PR **title** is the conventional commit that will appear in the merge commit: `feat(parser): support setext headings`.
3. The PR **body** follows the template in `.github/PULL_REQUEST_TEMPLATE.md`. Mandatory items:
   - **What**: clear, concise description.
   - **Why**: problem statement + link to issue (`Fixes #123`).
   - **Test evidence**: paste the relevant test run output / E2E screenshot.
   - **Breaking changes** section with migration instructions if `!` is used.
4. Labels: `phase-N`, `area: <scope>`, `bug`/`feature`/`docs`/`chore`.

### During Review

- Expect questions. Assume good intent. Answer every comment; `👍` without action is not an answer.
- Push fixup commits during review. Maintainers will squash on merge; do not rewrite history yourself mid-review unless asked.
- If the PR grows beyond its original scope, split it. Maintainers will ask for this explicitly.

### Merge

Merge strategy (maintainer action):

- **Squash merge** into `develop` for most PRs. The final commit message is the PR title + body (conventional).
- **Rebase merge** only for multi-commit PRs where each commit stands alone and is clean.
- CI must be green. At least one approving review from a CODEOWNERS entry. No unresolved discussions.

---

## Reporting Bugs & Requesting Features

- **Bugs** → use the [Bug Report template](../../issues/new?template=bug_report.md). Please include:
  - Minimal reproduction steps (not a 300-line log — bisect first).
  - Expected vs actual.
  - OpenBooklet version, OS, browser, Go/Node versions, provider used (Ollama / OpenAI / etc.).
  - `.obk` snippets that trigger the issue (sanitized if sensitive — do **not** paste real keys or sensitive runbook content).
- **Features** → use the [Feature Request template](../../issues/new?template=feature_request.md). Please state:
  - Which phase in PLAN.md this belongs to (or argue for a phase adjustment).
  - The user problem, not your proposed solution.
  - Why this can't be a template or third-party exporter/provider.

---

## Reviewer Expectations (for Maintainers)

1. **PLAN.md alignment first.** If the PR deviates from the spec and the spec wasn't updated, send it back for discussion.
2. **Tests before style.** If the tests are missing or weak, block on that — you can always auto-format later.
3. **No vendor leakage.** Confirm no vendor SDK import appears in `internal/`.
4. **Round-trip fidelity.** If parser or serializer changed, the critical round-trip tests must be extended, not just existing ones passing.
5. **No secret logging.** Confirm all new config values that hold credentials use `config.Secret` and pass the redaction test.

---

## Governance & Decisions

- **Spec changes** (anything in PLAN.md that alters a product behavior) → Discussion → maintainer consensus → update PLAN.md → then implement.
- **Architecture changes** → same: update ARCHITECTURE.md (and add/update an ADR under `docs/adr/`) with the decision and rationale, then implement.
- **Implementation details** → PR review + CI green is sufficient.

Consensus is sought among the maintainers. A maintainer vote is **lazy majority**
of responding maintainers within 7 calendar days; tie votes or blocked decisions
are escalated to the project leads.

### Project Leads & Maintainers

OpenBooklet was created and is maintained by:

| Handle | Role |
|--------|------|
| `marcuwynu23`  | Creator, Project Lead, Maintainer |
| `iammwwhobuild` | Creator, Project Lead, Maintainer |

Disagreements between contributors and maintainers that cannot be resolved
through discussion are resolved jointly by the project leads after fair
consideration of all arguments. Either project lead can request that the
question be reopened at any time with new evidence. No decision is final
forever; better data or a clearer user pain always reopens the discussion.

CODEOWNERS (forthcoming in `.github/CODEOWNERS`) will list both project leads
as default owners for all paths, with per-package domain reviewers added as
the codebase grows.

---

## Credits

Contributors will be listed in the release notes for each version and in the `AUTHORS` file once it exists. Significant design contributions are also noted inline in ARCHITECTURE.md ADR footers.

---

*Thanks again. The next time someone follows an SOP you helped build at 3 AM, you'll have made their night shorter.*
