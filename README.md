<div align="center">

  <img src="docs/assets/banner.jpg" alt="Consize" width="800" />

# Consize

**Policy-governed infrastructure optimization with safe, verifiable execution.**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Development](https://img.shields.io/badge/v0.3.0-in_development-0aa174.svg)](https://github.com/consize-oss/consize)

</div>

## v0.3.0 Is in Development

Consize v0.3.0 is the next open-source release and is currently under active
development. It rebuilds Consize as a provider-neutral optimization control
plane: infrastructure evidence becomes an explainable recommendation, policy
and preflight checks decide whether it may proceed, and a durable safety loop
owns execution, verification, recovery and rollback.

The code on `main` is the development baseline for v0.3.0. It is not yet a
stable production release. The supported v0.2 line remains available from the
[`v0.2.0` tag](https://github.com/consize-oss/consize/tree/v0.2.0) and
[`release/0.2`](https://github.com/consize-oss/consize/tree/release/0.2).

## Features

- **Kubernetes optimization:** discover workloads and identify opportunities to
  reduce unnecessary CPU and memory reservations.
- **Evidence-backed recommendations:** use real Prometheus usage and health data
  to explain what should change and why.
- **Safety headroom:** retain configurable capacity above observed demand and
  limit how much a resource can be reduced in one step.
- **Policy guardrails:** define when changes may proceed, require approval or be
  blocked based on environment, risk and available evidence.
- **Dry runs and reviewable plans:** inspect a proposed change and its safety
  checks before anything is applied.
- **Controlled remediation:** apply approved Kubernetes changes through one
  governed action path.
- **Post-action verification:** monitor workload health after a change and
  confirm that configured safety conditions remain satisfied.
- **Automatic rollback:** restore the previous resource configuration when
  verification fails or the action cannot complete safely.
- **Restart recovery:** continue pending action and verification work after the
  Consize process restarts.
- **Audit history:** retain the recommendation, policy decision, action,
  verification result and rollback outcome.
- **Extensible integrations:** support additional infrastructure, metrics, cost
  and action providers through the Consize plugin model.
- **Clear savings estimates:** show projected savings separately from savings
  confirmed by provider billing data.

## Current MVP Scope

The working optimization path currently targets Kubernetes Deployments using
Prometheus evidence. The resource and plugin contracts are provider-neutral,
but AWS, GCP, databases and other resource types are not automatically
discovered or remediated yet.

The following capabilities are planned after the Kubernetes OSS foundation is
qualified:

- additional cloud and database discovery/action plugins;
- GitHub-hosted public plugin catalog and publisher trust distribution;
- billing exports, FOCUS normalization and realized-savings reconciliation;
- GitOps pull-request workflows and additional notification integrations;
- scheduled reporting and broader multi-provider optimization algorithms.

## Safety Model

Every infrastructure mutation must follow one controlled path:

```text
discovery
  -> evidence
  -> recommendation
  -> authorization and policy
  -> preflight checks
  -> durable action intent
  -> apply
  -> rollout readiness
  -> metrics verification
  -> verified completion or rollback
  -> durable audit history
```

Review and dry-run operations may produce a plan, but they cannot mutate
infrastructure. Approved execution is handled by the durable safety controller.

## Documentation

Read the product and engineering documentation at
[docs.consizehq.com](https://docs.consizehq.com).

Development references in this repository:

- [`DEVELOPMENT.md`](DEVELOPMENT.md)
- [`local-lab/README.md`](local-lab/README.md)
- [`docs/v0.3-foundation-reference.md`](docs/v0.3-foundation-reference.md)

## Release Lines

- `main`: active v0.3.0 development; new work uses short-lived branches and PRs.
- `release/0.2`: supported maintenance line for v0.2.x fixes.
- `v0.2.0`: immutable source reference for the published v0.2.0 release.

See [`docs/maintenance-0.2.md`](docs/maintenance-0.2.md) for the v0.2 maintenance
and hotfix process.

## Contributing

Contributions are welcome. Please open an issue or discussion before large
architecture changes, keep provider-specific behavior behind plugin contracts,
and include tests for the safety behavior affected by a change.

## License

Consize is licensed under the [Apache License 2.0](LICENSE).
