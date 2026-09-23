# Consize Roadmap

Consize ships narrow, then widens. **Consize v0.3.0 is in active development on `main`**, building a safer, extensible foundation for turning infrastructure evidence into governed optimization actions.

## Current status

- **Stable:** [`v0.2.0`](https://github.com/consize-oss/consize/tree/v0.2.0), maintained on [`release/0.2`](https://github.com/consize-oss/consize/tree/release/0.2).
- **In development (`main`):** v0.3.0, described below. The first workflow targets Kubernetes Deployments and Prometheus. Broader resource types (databases, data warehouses, storage, CI/CD, GPU/LLM spend) and additional cloud providers follow once this foundation is qualified.

## v0.3.0 Features

- **Evidence-backed recommendations:** use real workload metrics to explain what should change and why.
- **Policy guardrails:** require approval, allow safe automation or block a change based on environment, risk and available evidence.
- **Safety headroom:** retain configurable capacity above observed demand and limit the size of each optimization step.
- **Dry runs and reviewable plans:** inspect proposed changes before they affect infrastructure.
- **Controlled Kubernetes remediation:** apply approved resource changes through one governed action path.
- **Post-action verification:** monitor workload health after every applied change.
- **Automatic rollback:** restore the previous configuration when verification fails or an action cannot complete safely.
- **Restart recovery:** continue pending actions and verification after Consize restarts.
- **Durable audit history:** retain recommendations, policy decisions, actions, verification results and rollback outcomes.
- **Extensible plugins:** connect additional infrastructure, metrics, cost and action providers through a common plugin model.
- **Clear savings reporting:** keep projected savings separate from savings confirmed by provider billing data.

## How to follow along or weigh in

- Track progress on `main` against the features above.
- Open a GitHub Discussion for larger architectural proposals; significant decisions are logged as ADRs (see [`docs/decisions.md`](docs/decisions.md)).
- See [`CONTRIBUTING.md`](CONTRIBUTING.md) for how to pick up a `good first issue`.