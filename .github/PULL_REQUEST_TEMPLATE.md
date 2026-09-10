# Pull Request

<!--
  Thanks for the PR! Please complete this template. PRs that do not follow
  [CONTRIBUTING.md](../../CONTRIBUTING.md) will be put on hold until fixed.

  Quick checklist before submitting (this is automated in CI, but do it locally first):
    - `make check`                (Go lint + vet + staticcheck + tests)
    - `cd web && npm run all-checks`  (tsc, eslint, vitest)
    - conventional commit title, signed-off-by (`git commit -s`), changelog entry

  The PR TITLE must be a single conventional commit line:
    feat(parser): support setext underlined headings
    fix(booklet): preserve section ordering on reload  (#123)
    docs(ARCHITECTURE): expand storage section + ADR 0002
-->

## 1. Summary

Describe what this PR does **and why**.

<!--
  Keep it user-focused. Example:
    > Implement Phase 4 Markdown parser support for setext-style (`---` / `===`)
    > underlined headings per CommonMark. Users importing existing READMEs that
    > use setext headings now get proper Section cells instead of losing the
    > heading level. Fixes #42.
-->

**What changed:**
**Why it was needed:**

---

## 2. Related Issue / Discussion

- **Closes:** #___   (issue this PR closes — auto-closes on merge)
- **Fixes:**  #___   (bug this PR fixes)
- **Related:** #___, #___, discussion URL, spec sections in PLAN.md:

  - PLAN.md §___
  - ARCHITECTURE.md §___
  - AGENTS.md §___

---

## 3. Type of Change

Mark the relevant option(s) AND **confirm the conventional-commit type of the PR title**
matches. (The PR title is the squash-merge commit message.)

- [ ] `feat` — New user-facing feature (MINOR SemVer bump)
- [ ] `fix` — User-visible bug fix (PATCH SemVer bump)
- [ ] `perf` — Performance improvement, no behavior change (PATCH)
- [ ] `refactor` — Code restructure, no feature/bug change
- [ ] `test` — Adding or fixing tests only
- [ ] `docs` — Documentation, README, ARCHITECTURE, ADR, Go doc comments
- [ ] `style` — Pure formatting / whitespace (should be batched with an actual change)
- [ ] `build` — Makefile, deps, Dockerfile, go.mod, package.json
- [ ] `ci` — GitHub Actions, CI config
- [ ] `chore` — Housekeeping, local tool config
- [ ] **BREAKING CHANGE** — `!` used in title + `BREAKING CHANGE:` footer in the commit. Requires MAJOR bump (or MINOR in 0.x); see [CONTRIBUTING.md § Breaking Changes](../../CONTRIBUTING.md#breaking-changes).
- [ ] Other (please describe):

---

## 4. PLAN.md & Architecture Alignment

OpenBooklet has strict phase discipline (see [CONTRIBUTING.md § Phase Discipline](../../CONTRIBUTING.md#phase-discipline)).

- **Targeted roadmap phase:** Phase ___ (or `spec-change / bugfix / maintenance`)
- **Does this change modify product behavior described in PLAN.md?**
  - [ ] No — pure implementation detail, bug fix, doc, refactor, test
  - [ ] Yes — a PLAN.md update PR is linked here: #___  (required before merge for behavior changes)
- **Does this change modify architecture in ARCHITECTURE.md?**
  - [ ] No
  - [ ] Yes — an ARCHITECTURE.md update or new ADR is in this PR / linked here: ADR-####
- **Does this PR introduce any vendor SDK import into `internal/`?**
  - [ ] No ✅  (required. Vendor SDKs go in `providers/<name>/` only, behind the `Provider` interface.)
  - [ ] Yes — explain below:

---

## 5. Testing

Describe the tests you ran. Paste evidence.

### 5.1 Tests added/updated

- [ ] **Unit tests** added for new logic
- [ ] **Integration tests** updated (SQLite `:memory:` or test FS)
- [ ] **API / httptest** coverage added for new endpoints
- [ ] **Provider contract tests** pass for affected providers (if provider-related)
- [ ] **Round-trip tests** updated if touching Markdown parser, `.obk` serializer/deserializer:
  - [ ] Markdown ⇄ Cells golden file added/updated
  - [ ] `.obk` ⇄ Model golden file added/updated
- [ ] **Context prompt-injection regression test** added (if changing context builder)
- [ ] **Streaming cancel / goroutine leak test** added (if changing LLM orchestration)
- [ ] **Status transition state-machine tests** updated (if adding statuses)
- [ ] **Frontend** — Vitest component/store tests added
- [ ] **Frontend** — Playwright E2E added or updated for golden-path changes

### 5.2 Local test run evidence

Paste the tail of `make check` + `cd web && npm run all-checks` output:

```
# paste here
```

### 5.3 Manual / exploratory steps performed

(e.g. "Ran `openbooklet serve --dev`, created a booklet via chat prompt, cells appeared,
exported Markdown round-tripped through `openbooklet import → export` without diff.")

---

## 6. Round-Trip & Critical-Path Sanity Checks

If your change touches ANY of the following areas, confirm the corresponding critical
tests still pass *and* that golden files were regenerated intentionally (not accidentally
— run with `-update` only when the contract truly changed).

| Area | Touched? | Golden file updated? | Intended change (Y/N) |
|------|:--------:|:--------------------:|:---------------------:|
| Markdown parser (md ↔ sections) | ☐ | ☐ | ☐ |
| `.obk` serializer / deserializer | ☐ | ☐ | ☐ |
| Context builder ordering / separators | ☐ | ☐ | ☐ |
| Provider interface / contract | ☐ | ☐ | ☐ |
| Status enum / state machine | ☐ | ☐ | ☐ |
| Booklet / section hierarchy / ordering | ☐ | — | ☐ |
| Export formats (md / html / …) | ☐ | ☐ | ☐ |

---

## 7. Screenshots / Demo (UI or output changes)

For web UI, CLI output, or `.obk` format changes, paste a screenshot or diff of the
before/after so reviewers can see the user-visible impact.

---

## 8. Breaking Changes

If `BREAKING CHANGE` is checked above, fill this in:

- **Affected API surface / interface / format:**
  - e.g. `.obk` v2, removed `GET /api/v1/booklets/{id}/sections` inline array behavior, new required `Provider` method
- **Migration path for users / downstream consumers:**
- **Migration doc / docs/migrations/NNNN-*.md:** (link or paste excerpt)
- **For 0.x — is this breaking change worth the churn?** Justify.

---

## 9. Security & Privacy

- [ ] No secrets / keys / tokens appear in code, tests, test fixtures, logs, or screenshots.
- [ ] Any new configuration fields that hold credentials use `config.Secret` (not `string`) and pass the redaction test.
- [ ] Any new file-path inputs are validated against path-escape / directory-traversal.
- [ ] Context builder still enforces strict TrustLevel ordering; no structural (Class A) prompt injection new capability introduced.
- [ ] If this could have security implications: SECURITY.md private-disclosure process was followed instead of / alongside this public PR.

---

## 10. CHANGELOG / Sign-off

- [ ] An entry has been added to the **`[Unreleased]`** section of [CHANGELOG.md](../../CHANGELOG.md) under the correct heading (`Added`, `Changed`, `Fixed`, `Security`, …).
- [ ] **Every commit** in this PR has:
  - [ ] Conventional commit format
  - [ ] DCO **`Signed-off-by:`** trailer (`git commit -s`). See [CONTRIBUTING.md § DCO](../../CONTRIBUTING.md#dco--sign-your-work).
  - [ ] No WIP / vague messages. Squashed if needed before merge.

---

## 11. Additional Notes

Anything else reviewers should know: dependencies, tricky edge cases, alternative approaches
considered, follow-up work planned.

---

*Reviewers, before approving:*
- [ ] PR title is a conventional commit.
- [ ] PLAN.md / ARCHITECTURE.md alignment confirmed or spec-change PR linked.
- [ ] No vendor SDK in `internal/`.
- [ ] Round-trip golden changes are intentional and documented.
- [ ] Error paths tested.
- [ ] Security checklist green.
- [ ] CHANGELOG entry present, DCO green, CI green.
