# Consize Roadmap

Consize ships narrow, then widens. The goal is to prove Consize can safely reduce Kubernetes waste, with clear evidence, approval, verification, rollback, and audit, before expanding into a broader plugin-driven optimization platform.

**Order of operations:** safe Kubernetes action → policy & plugin framework → multi-cluster and enterprise readiness → broader resource ecosystem (databases, data warehouses, storage, CI/CD, GPU/LLM spend).

## Principles

- **The MVP stays narrow.** Prove Consize can safely reduce Kubernetes waste before chasing breadth.
- **The plugin shape comes early.** Even where early integrations are internal adapters, they're built behind plugin-shaped contracts from day one.
- **Trust before automation.** Auto-apply is not the MVP's main proof; earning trust is.
- **Realized savings are tracked separately from estimated savings**, from day one. A change only counts once it's verified.

## Current status

- **Stable:** [`v0.2.0`](https://github.com/consize-oss/consize/tree/v0.2.0), maintained on [`release/0.2`](https://github.com/consize-oss/consize/tree/release/0.2).
- **In development (`main`):** v0.3.0, the plugin-based optimization foundation described below (Phases 0–4). First workflow targets Kubernetes Deployments and Prometheus.

## Phases

### Phase 0: Foundation and Strategy

Clarify Consize's category, architecture, and MVP scope: product positioning, target customer, plugin contract draft, policy model draft, MVP UI, demo narrative.

**Done when:** the one-sentence positioning is clear, the MVP can be explained in under a minute, and the team is aligned on the safety-first thesis.

### Phase 1: MVP — Safe Kubernetes Recommendations

Find Kubernetes waste and explain the safest next action, using CPU/memory requests & limits, Prometheus metrics, basic cloud pricing, and a conservative p95 rightsizing algorithm on a single cluster.

**Out of scope:** marketplace, external plugin runtime, multi-cloud billing normalization, AI spend optimization, automatic production changes by default.

**Done when:** every recommendation has evidence, a retained safety buffer, and a clear policy decision (blocked, approval-required, PR-based, or auto-allowed).

### Phase 2: MVP — Approval, GitOps, and Audit

Turn recommendations into reviewable changes: GitHub PR action, approval workflow, audit events, reject/snooze with reasons, policy version attached to each recommendation.

**Done when:** production recommendations require approval, users can open a PR instead of applying directly, and the audit trail shows who acted and why.

### Phase 3: MVP — Guarded Apply and Verification

Apply a safe step and verify the result: step-limited direct apply, pre-apply state capture, Prometheus (and optionally Datadog) verification window, rollback on failure, realized savings counted only after a pass.

**Done when:** Consize never applies without audit or outside policy, rollback restores the previous configuration, and failed verification produces evidence.

### Phase 4: Plugin Framework MVP

Expose internal integrations as plugins: manifest schema, plugin health page, permissions, internal registry. Initial plugins: Prometheus Metrics, Kubernetes Resources, Kubecost/OpenCost Cost Data, Datadog Metrics, p95 Rightsizer, Conservative Rightsizer, GitHub PR, Kubernetes Apply.

**Done when:** plugins declare capabilities and show status in the UI, read-only plugins can't mutate, action plugins require policy approval, and every recommendation shows which plugin/algorithm produced it.

### Phase 5 and beyond: Policy-as-Code and Ecosystem Expansion

Make trust configurable and auditable (environment and team policies), then widen beyond Kubernetes: databases, data warehouses, storage, CI/CD, GPU/LLM spend, and additional cloud providers, once the OSS foundation is qualified.

## How to follow along or weigh in

- Track progress on `main` against the phases above.
- Open a GitHub Discussion for larger architectural proposals; significant decisions are logged as ADRs (see [`docs/decisions.md`](docs/decisions.md)).
- See [`CONTRIBUTING.md`](CONTRIBUTING.md) for how to pick up a `good first issue`.