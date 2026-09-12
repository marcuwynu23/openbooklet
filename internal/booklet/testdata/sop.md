---
title: Kubernetes Deployment SOP
audience: l1-operations
---

This document is generated for parser testing.
It spans multiple lines before the first heading.

# Kubernetes Deployment SOP

Intro text with a [link](https://example.com/k8s) and an image:

![cluster diagram](cluster.png)

## Purpose

Explain the purpose of this SOP.

### Scope

In scope:

- Kubernetes cluster access
- kubectl
- Required permissions

Out of scope:

1. Cloud account provisioning
2. Network design

## Prerequisites

Generate the required prerequisites.

| Tool    | Version |
| ------- | ------- |
| kubectl | >= 1.29 |
| helm    | >= 3.14 |

> Note: lines starting with `#` inside a quote are body text.
> # not a heading

    # indented code is body text, not a heading

```bash
# comments inside fenced code are body text
kubectl apply -f deployment.yaml
## also not a heading
```

~~~mermaid
flowchart TD
    A[Check cluster] --> B[Deploy]
    B --> C[Verify]
~~~

## Deployment

Run the deployment, then verify.

### Verify

```bash
kubectl rollout status deployment/web
```

# Appendix

Closing hashes normalize away. Setext-style underlines below are body text.

Not a heading
---
