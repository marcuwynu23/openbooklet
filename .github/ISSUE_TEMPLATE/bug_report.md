---
name: Bug Report
about: Report something that doesn't work correctly in OpenBooklet
title: "bug(<scope>): <short description>"
labels: ["bug", "triage"]
assignees: []
projects: []
---

# Bug Report

Thank you for taking the time to report a bug. A good report helps us reproduce, isolate,
and fix quickly. Please fill out the information below.

**PLEASE NOTE: Do NOT include secrets, credentials, API keys, or production `.obk` content
in this report. Replace with placeholders like `sk-REDACTED` and upload a **sanitized** `.obk`
if needed. For security vulnerabilities, see [SECURITY.md](../../SECURITY.md) — do NOT open a
public bug report.**

---

## 1. Bug Description

A clear and concise description of what the bug is.

<!--
  Good: "When I add a `###`-level child cell under a `##` section and then reorder the
  parent above another `##` section, the child cell's ParentID is not updated and on
  next reload it renders as an orphan at the top level."

  Bad:  "Reorder is broken."
-->

---

## 2. Steps to Reproduce

Provide a minimal, deterministic set of steps. If you can reproduce with a small `.obk`
or Markdown input, paste it inside a code block.

1. Go to '...'
2. Click on '...'
3. Scroll down to '...'
4. See error

```yaml
# paste the minimal .obk that triggers the bug, or "N/A"
version: "1"
booklet:
  id: repro-case
  title: Repro
```

```markdown
# or paste the minimal Markdown input, or "N/A"
# Title
## Section 1
### Subsection A
content
```

---

## 3. Expected Behavior

What did you expect to happen?

---

## 4. Actual Behavior

What actually happened? Paste error messages, stack traces, or screenshots here.

```
Paste error text / stack trace / logs here. Redact keys or hostnames.
```

---

## 5. OpenBooklet Environment

**Please fill all that apply.**

- **OpenBooklet version / commit SHA:** (e.g. `v0.1.0`, commit `01H…`)
- **How was it run?** (source `make build` / release binary / Docker / `npm run dev` only)
- **CLI command or URL path:** (e.g. `openbooklet serve --dev`, UI route `/booklet/01H…/edit`)
- **OS:** (e.g. Windows 11 23H2, macOS Sonoma 14.5, Ubuntu 24.04)
- **Architecture:** (amd64 / arm64 / other)
- **Go version** (if built from source): `go version` →
- **Node / npm version** (if dev UI): `node -v` → ___ ; `npm -v` → ___
- **Browser** (if UI bug): (e.g. Chrome 125, Firefox 126, Safari 17)
- **Device form factor:** desktop / tablet / mobile

**LLM Provider (if AI-operation related bug):**

- **Provider name:** (ollama / openai / anthropic / gemini / compatible / other)
- **Model:** (e.g. `llama3`, `gpt-4o-mini`)
- **Local or remote:** (local, self-hosted, cloud vendor)
- **Is the provider reachable at the time of the bug?** (Yes / No / Partial)

---

## 6. Affected Area

Check all that apply:

- [ ] **Markdown parser** (md → cells, headings, code blocks, tables, mermaid, lists)
- [ ] **`.obk` serialize/deserialize** (round-trip loses fields, YAML errors)
- [ ] **Booklet / section hierarchy & ordering** (parent/child, reorder, orphan)
- [ ] **Cell editing** (Markdown editor, preview, split view, undo/redo)
- [ ] **AI operations** (generate, regenerate, rewrite, expand, shorten, review, cancel)
- [ ] **Streaming (SSE)** (content not appearing, hangs, cancel not working)
- [ ] **Context engine / references** (`@section:id`, `@file`, prompt injection boundaries)
- [ ] **Templates** (builtin or custom YAML templates not applying correctly)
- [ ] **Storage / persistence** (SQLite, filesystem, save/reload loses data)
- [ ] **Export / Import** (Markdown, HTML, PDF later)
- [ ] **CLI** (subcommands, flags, output formatting)
- [ ] **REST API** (endpoints, status codes, envelopes, validation errors)
- [ ] **Web UI** (sidebar, toolbar, routes, state, forms)
- [ ] **Review engine** (findings, false positives, silent edits)
- [ ] **Git integration** (Phase 13 — repo detection, diff, commit)
- [ ] **Security** (secrets leaking, path traversal, XSS, injection — if so, you should use PRIVATE reporting instead!)
- [ ] **Other:** describe below

---

## 7. Severity (your assessment)

Pick the **highest** applicable:

- [ ] **Critical** — System unusable, data loss/corruption, security exploit, crash on every open
- [ ] **High** — Major feature broken, no workaround, blocks the MVP golden path
- [ ] **Medium** — Feature partially broken, workaround exists, no data loss
- [ ] **Low** — Minor UI glitch, wording issue, cosmetic, no functionality impact

---

## 8. Additional Context

Anything else that might help:

- First known good version (if regression):
- Workarounds you've found:
- Related issues / discussions / links:
- If you know the fix: paste a diff or link to your fork branch below (or just open a PR linked to this issue!)

---

## 9. Possible Solution (optional)

If you have ideas on how to fix it, describe them here. Or: "I'd like to work on this — please assign to me."

---

*Before submitting, please confirm:*
- [ ] I have searched existing open/closed issues and this is not a duplicate.
- [ ] I have read [CONTRIBUTING.md](../../CONTRIBUTING.md) and understand the PR process if I open a fix.
- [ ] No secrets, credentials, or sensitive operational data is included in this report.
- [ ] If this is a security vulnerability, I have NOT opened a public report and have instead used the private disclosure process in [SECURITY.md](../../SECURITY.md).
