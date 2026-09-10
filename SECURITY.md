# Security Policy

OpenBooklet processes operational documentation. That content may contain sensitive information: infrastructure topology, credentials-in-prose, rollback procedures, incident playbooks. The project takes security seriously at every layer: **the local-first promise**, **prompt-injection boundaries**, **secret redaction in structured logs**, and a clear, private disclosure path for anything you find.

Read this in full before opening a security report, a public bug report, or a PR titled "fix: secure thing".

---

## Supported Versions

OpenBooklet follows **Semantic Versioning**. Security fixes are back-ported to the last **two** MINOR releases on the current MAJOR line. Once a new MAJOR release is cut, the previous MAJOR line receives fixes for 6 months, then reaches end-of-life.

During the pre-1.0 period (Phase 0 through Phase 15), only the **latest release** receives security updates. Early adopters on pre-1.0 are expected to upgrade promptly.

| Version | Supported | Notes |
|---------|:---------:|-------|
| **0.x (current)** | ✅ | Pre-1.0 — only the latest tagged release. Upgrade to stay supported. |
| 1.x (future) | ✅ | Planned post-Phase 16 release. Two latest minors supported. |
| Everything else | ❌ | Including any forks, self-modified builds, or `-dev` / `-nightly` snapshots. |

If you are on an unsupported version and find a vulnerability, please still report it. We'll triage and fix it if it affects supported versions, but we will not back-port to unsupported releases.

---

## Reporting a Vulnerability

**DO NOT open a public GitHub issue.** Public disclosure — even a vague one — puts every OpenBooklet user at risk before a patch is ready.

### How to Report (pick ONE)

1. **GitHub Private Vulnerability Report (preferred):**
   - Navigate to the repository → **Security** → **Report a vulnerability**.
   - This creates a private, CVE-number-ready draft shared with both maintainers (`marcuwynu23`, `iammwwhobuild`).
   - Reproducer artifacts (`.obk` files, logs, screenshots) can be attached directly.
2. **Encrypted email:**
   - Write to the security contacts published in the project's `SECURITY_CONTACTS` file (if present) or request the address by DMing either maintainer on GitHub.
   - If you need PGP/GPG keys, ask for them in an unencrypted email/DM first; do not send vulnerability details until keys are exchanged.
3. **Escalation (if no response in 48 hours):**
   - Send a follow-up GitHub private message individually to **both** maintainer accounts (`marcuwynu23` and `iammwwhobuild`), then escalate to the GitHub organization owner listed on the repository profile page.

### What to Include

A good report lets maintainers reproduce, confirm, and fix in one pass. Please provide:

- **Summary** — one paragraph.
- **Affected versions** — exact tag or commit SHA, build method (source / release binary / Docker).
- **Environment** — OS, Go version / Node version, provider used (Ollama, OpenAI, …), local or self-hosted.
- **Reproduction steps** — minimal and deterministic. A one-line Go test that triggers the issue is the gold standard.
- **Proof of impact** — what an attacker can actually do (read, write, exfiltrate, execute, prompt injection outcome, etc.).
- **Your assessment of severity** (CVSS v3.1 vector string is helpful, but not required).
- **Suggested fix** (if you have one — optional, but always welcome).

**Do not include:**
- Real API keys, passwords, `.obk` files containing proprietary data, production logs, personal data of third parties, or PII of any kind. Replace with placeholders (`sk-REDACTED`, `example.obk`).

---

## Disclosure Timeline — Our Commitment

| Step | SLA | Notes |
|------|-----|-------|
| Acknowledge receipt of report | **2 business days** | Automated or human reply. |
| Initial triage & severity assessment | **5 business days** | We may ask clarifying questions. |
| Fix developed & validated | Varies by severity | Critical ≤ 14 days; High ≤ 30 days; Medium/Low ≤ 90 days. |
| Patch released & advisory published | On release date | Coordinated with reporter to credit. |
| Public disclosure | On release date **or** 90 days from receipt, whichever comes first. | We will request an extension only if the fix is unusually complex and you agree. |

We follow the industry-standard **90-day disclosure window** from report receipt. Exceptions are negotiated explicitly and in writing.

---

## Scope — What Counts as a Security Vulnerability?

### In-Scope

- **Remote or local code execution** in the host process via parsing a malicious `.obk`, Markdown, or file attachment.
- **SQL injection / directory traversal / path escape** through any user input (booklet id, section id, file names, export paths).
- **Secrets leakage** into logs, crash dumps, error messages, `.obk` export output, or URL query parameters.
- **Prompt-injection that actually elevates external file content to LLM instructions** when the context builder ordering is violated. (See PLAN.md §42.) Finding that "a model *might* listen to the file" is not a vuln; finding that the delimiters were missing and the model actually obeyed *is*.
- **Authentication / authorization bypass** in the self-hosted server (once auth ships post-MVP).
- **XSS / CSRF** in the bundled web UI that affects users viewing untrusted booklets.
- **Arbitrary file read/write** as the OpenBooklet process user via import/export or file-context features.
- **Data loss / corruption** in `.obk` or SQLite triggered by standard operations (save, undo, version restore).

### Out-of-Scope

- **Vulnerabilities that require the attacker to already have local code execution or modify the binary.** Local-first software assumes the host is trusted.
- **Model "hallucinations" or factually incorrect AI output.** AI output is non-authoritative; see PLAN.md §20 (Human-in-the-Loop). Reports about this go to regular issues, not security.
- **Prompt injection in content the *user themselves typed* into their own prompt box.** The user is the authority in a local-first app; injection only matters across the trust boundary (external files, imported `.obk` from strangers).
- **Dependencies with no known exploit path** where the vulnerable code is unreachable in OpenBooklet. Open a regular issue / dependabot PR — we still want to fix it, but it doesn't need the private security process.
- **Theoretical issues with no PoC.** "Maybe someone could…" without a working PoC opens as a regular discussion/issue first. Provide the PoC and we'll reclassify as needed.
- **Social engineering of users.** Teach them in docs; don't file a vuln.

---

## Prompt Injection Policy

Prompt injection is a domain-specific concern for OpenBooklet. We define two classes:

### Class A — Structural (Security Vulnerability)

The *context builder* misplaces content in the ordered hierarchy (System > App > User > Booklet Instr > Section Instr > Booklet Content > External Files > External Refs), allowing lower-trust content to appear above a higher-trust instruction separator. **Report via private security disclosure.**

### Class B — Model-Behavioral (Regular Issue)

Everything is correctly ordered and separated, yet the LLM *still* chooses to obey the file over the instructions due to its own training weights. This is an LLM behavior issue, mitigated through instruction engineering, stronger delimiters, and explicit "content blocks" in the system prompt. **Report as a regular issue with the `prompt-injection` label.**

All structural Class A fixes come with a regression test in the context-builder test suite asserting the assembled prompt hierarchy byte-for-byte.

---

## Secret Handling & Best Practices for Users

This section is in the security policy because 90% of real-world "security bugs" reported are user misconfiguration. Before you file, please confirm:

1. **API keys live in env vars or the user-level config file** (`~/.config/openbooklet/config.yaml` on Unix, `%AppData%\openbooklet\config.yaml` on Windows). **Never** paste keys into a Booklet, a cell prompt, a section, or commit `config.yaml` to a repo.
2. **`.obk` files are plain YAML.** Treat them as data. If a booklet contains sensitive operational details, don't upload it to public repos, paste it into bug reports, or host it on untrusted shared drives without encryption at the filesystem level.
3. **The `--dev` flag enables plain HTTP.** Only use it on localhost. In any non-local deployment, put OpenBooklet behind TLS (nginx, Caddy, or the upcoming built-in `--tls-*` flags).
4. **Don't mix trust levels in one Project context.** If you attach a `.obk` from an untrusted source, its content is treated as low-trust data — but don't open it on the same instance handling production keys if you can avoid it.

Violations of these best practices are user-education issues, not security vulnerabilities in OpenBooklet. We'll gladly take PRs that add runtime warnings when the app detects them.

---

## Fix Process — For Maintainers

1. Reproduce locally. Write the failing test first.
2. Develop the fix on a **private fork or branch** — no public PR until release day.
3. Extend the test suite with regression tests.
4. Add a `CHANGELOG.md` entry under `### Security` in `[Unreleased]`.
5. Draft the GitHub Security Advisory with CVE (request one via the advisory form if eligible).
6. Cut a PATCH release containing *only* the security fix and its tests. No unrelated feature changes.
7. Publish the advisory, the release, and email the reporter a thank-you + credit.
8. Back-port to the previous supported MINOR line per the table above.

---

## Hall of Fame

Reporters who privately disclose valid, confirmed, and fixed vulnerabilities will be credited by name (with permission) in:

- The GitHub Security Advisory credits field.
- The corresponding release notes.
- The `docs/SECURITY-ACKS.md` file once the first report lands.

No bounties are offered during the pre-1.0 period, but post-v1.0 a bounty program (at least for Critical and High) is on the roadmap.

---

## Questions

If you are unsure whether something qualifies, err on the side of reporting privately. We would much rather spend an hour triaging a false positive than miss a real vulnerability. The worst we'll say is "thanks, please open that as a regular issue."
