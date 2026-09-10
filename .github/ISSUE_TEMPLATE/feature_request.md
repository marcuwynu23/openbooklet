---
name: Feature Request
about: Suggest an idea, enhancement, or new capability for OpenBooklet
title: "feat(<scope>): <short description>"
labels: ["enhancement", "triage"]
assignees: []
projects: []
---

# Feature Request

Thank you for suggesting an idea to improve OpenBooklet. The more context you give, the
better we can evaluate whether (and in which phase) your suggestion belongs.

**Before filing:**
- Read [PLAN.md](../../PLAN.md). Many features are already planned! State which existing
  section your request relates to (if any).
- Check open issues & discussions to avoid duplicates.

---

## 1. Feature Summary

One paragraph: what is the user-facing capability you want?

<!--
  Good: "Support for linking to anchors inside external Markdown files via a new
  `@file:README.md#section-id` reference syntax in the context builder, so that
  when I attach a file I can scope context to a subsection."

  Bad:  "Add file support."
-->

---

## 2. Problem Statement

What user problem does this solve? What is the pain or limitation you hit today,
with a concrete use case?

*As a `<persona>`, I want to `<action>`, so that `<outcome/why>`.*

---

## 3. Proposed Solution

Describe how you imagine this working from the user's perspective. Screenshots, mockups,
or ASCII diagrams are helpful!

```
# Example mockup of a UI change:

+--------------------------------------------------+
| [ Attach file ] [ Link anchor ] [ @ref:help  ]   |
+--------------------------------------------------+
```

If you have a technical design in mind (new API endpoint, new Go package/interface,
new config schema, etc.), briefly sketch it:

- Backend: package, interface, or schema changes
- Frontend: new component, new route, new store slice
- `.obk` schema: new top-level or section-level fields
- Provider / Exporter / Template interface extensions

---

## 4. Alternative Solutions / Workarounds

Have you considered alternative approaches? What manual workaround are you using today?

---

## 5. Alignment with PLAN.md & Phases

Reference the closest existing section in [PLAN.md](../../PLAN.md):

- **Closest phase from PLAN.md §47–§68:** Phase ___ (or "new phase needed")
- **MVP (v0.1) required?** (see [README.md MVP Golden Path](../../README.md#mvp---golden-path-v01)):
  - [ ] Yes — this blocks the 9-step chain
  - [ ] No — useful enhancement, post-MVP
- **Relates to which built-in principle?** (pick 0–3)
  - [ ] Local-first
  - [ ] AI-optional
  - [ ] Provider-agnostic
  - [ ] Human-controlled
  - [ ] Markdown-native
  - [ ] Git-friendly
  - [ ] Structured documents
  - [ ] Extensible
  - [ ] Open-source
  - [ ] Self-hostable

---

## 6. Expected Impact

- **Primary audience:** (Authors / DevOps / SREs / Technical Writers / Managers / Maintainers)
- **Benefit:** Saves time / prevents mistakes / enables new use case / improves UX / reduces cost / other
- **Rough importance:** Critical / Nice-to-have / Long-term wish
- **Will you or your org benefit from this in the next 3 months?** Yes / No / Maybe

---

## 7. Example Use Case(s)

Walk through 1–3 concrete, real-world workflows.

> *"I'm writing an SOP for a Kubernetes deployment. My repo has `deployment.yaml` with a
>  40-line `spec.template.spec.containers[].envFrom[].secretRef` section. Today I can
>  attach the whole `deployment.yaml` as context, but I only need lines 120–160 to be in
>  the AI context for the 'Deployment Procedure' cell. With anchor references I can point
>  to the exact block and save tokens while avoiding unrelated context leaking in."*

---

## 8. Open Questions / Tradeoffs

Anything you're unsure about, or known tradeoffs of the approach:

- Does this require a `.obk` schema bump? (If yes, mark "breaking" in the PR.)
- Does this break provider abstraction? (No vendor-specific features in `internal/` please.)
- Does this make the MVP bigger? (If so, suggest a phased rollout.)

---

## 9. Suggested Implementation (optional)

If you want to contribute the implementation, sketch the plan here. Otherwise a maintainer
will triage and provide guidance.

---

*Before submitting, please confirm:*
- [ ] I have read [PLAN.md](../../PLAN.md) and searched issues for duplicates.
- [ ] I have stated the *user problem*, not just my proposed solution.
- [ ] For spec-level changes (product behavior changes), I understand that a discussion + PLAN.md update happen **before** code changes per [CONTRIBUTING.md Governance](../../CONTRIBUTING.md#governance--decisions).
- [ ] I would be willing to help implement this (yes / no / with guidance).
