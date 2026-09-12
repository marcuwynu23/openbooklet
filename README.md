# OpenBooklet

> **Chat is the creation interface. Sections are the editing interface. Markdown is the content layer. The Booklet is the document. AI is the assistant.**

OpenBooklet is an open-source, AI-native documentation workspace. You describe
what you want in a chat-style prompt; AI streams Markdown back; OpenBooklet
splits it into an editable hierarchy of sections. Each section owns its prompt,
Markdown content, status, generation metadata, and history. Booklets persist to
SQLite and to human-readable, Git-friendly `.obk` YAML files.

## Run It

Prerequisites: Go 1.24+, Node 22+.

```bash
make frontend-build   # build the web UI once
make run              # serve http://localhost:8080
```

Create a booklet, describe the document, and watch streamed Markdown turn into
editable sections — or write sections by hand, no AI required. With no provider
configured the editor works fully offline; AI features report `503 no_provider`.

```bash
# Optional: local-first AI via Ollama
export OPENBOOKLET_PROVIDER=ollama
export OPENBOOKLET_MODEL=llama3
```

See [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) for all environment variables,
Make targets, testing, and the HTTP API reference.

## Core Concepts

| Concept | Meaning |
|---------|---------|
| **Booklet** | The document: metadata, header/footer Markdown, instructions, sections. |
| **Section** | One numbered unit (`1. ## Title`): prompt + Markdown + status + history. |
| **Provider** | Pluggable LLM backend (`Provider` interface; Ollama + OpenAI-compatible). |
| **`.obk`** | YAML source of truth. Round-trip lossless, meaningful Git diffs. |

Design principles: local-first, AI-optional, provider-agnostic,
human-controlled, Markdown-native, Git-friendly. Full spec: [PLAN.md](PLAN.md).

## Project Status

Working slices through Phase 7 are implemented and tested. Phase checklist and
what's next: [ROADMAP.md](ROADMAP.md).

## Docs

| File | Purpose |
|------|---------|
| [ROADMAP.md](ROADMAP.md) | Phase checklist and upcoming work |
| [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) | Setup, testing, API reference |
| [PLAN.md](PLAN.md) | Full 75-section product specification |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Technical architecture and layering |
| [AGENTS.md](AGENTS.md) | Senior Engineer handbook for contributors and agents |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Workflow, commits, PR process |
| [SECURITY.md](SECURITY.md) | Security policy and disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release history |

## License

Apache License 2.0 — see [LICENSE](LICENSE).

```
Copyright 2026 OpenBooklet Contributors
```

*Built by and for the people who get paged at 3 AM.*
