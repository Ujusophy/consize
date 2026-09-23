<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="/docs/assets/logoreadme-white.png">
    <img width="300" src="/docs/assets/logoreadme-black.png" alt="consize">
  </picture>
</h1>

<p align="center">
  <b>The safe optimization control plane for cloud platform spend.</b>
</p>

<h4 align="center">
  <a href="https://github.com/consize-oss/consize/blob/main/docs/customer-guide.md">Docs</a> |
  <a href="#try-the-interactive-sandbox">Sandbox</a> |
  <a href="https://github.com/consize-oss/consize/blob/main/VISION.md">Vision</a> |
  <a href="https://github.com/consize-oss/consize/blob/main/SECURITY.md">Security</a>
</h4>

<h4 align="center">
  <a href="https://github.com/consize-oss/consize/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Consize is released under the Apache 2.0 license." />
  </a>
  <a href="https://consizetownhall.slack.com/">
    <img src="https://img.shields.io/badge/Community-Slack-4A154B.svg?logo=slack" alt="Consize community Slack" />
  </a>
</h4>

<img src="/img/demo-dashboard.png" width="100%" alt="Consize dashboard" />

**Consize** is the open source cost optimisation tool that turns cost and usage signals into governed, verifiable optimization actions. It finds waste, explains the safest change, routes it through policy and approval, applies it gradually, verifies health, rolls back on regression, and proves *realized* savings, not just estimated ones.

---

## Getting Started

| 🧪 Try the Sandbox | 🚀 Production Install |
| --- | --- |
| Run everything locally in one container, pre-seeded with data and a live regression to watch Consize catch and roll back. | Install onto a live cluster (AWS/GCP) with the official Helm chart. Read-only by default until you grant least-privilege `RoleBindings`. |

### Try the Interactive Sandbox

```bash
docker run -p 3000:3000 -p 8080:8080 -it ghcr.io/consize-oss/consize-sandbox:latest
```

Open `http://localhost:3000` and watch the Verifier catch an intentional regression on the `checkout-api` workload, then automatically roll it back to restore safely.

### Production Installation

```bash
# 1. Export the default values to customize your installation
helm show values oci://ghcr.io/consize-oss/charts/consize > values.yaml

# 2. Install the chart using your customized values
helm install consize oci://ghcr.io/consize-oss/charts/consize \
  --version 0.2.0 \
  --namespace consize-system \
  --create-namespace \
  -f values.yaml
```
Consize is built to run safely inside your cluster. It uses read-only access for analysis and requires explicit, least-privilege, namespace-scoped RoleBindings before it can apply any changes.

---

## Configuration

For detailed instructions on configuring Helm values, setting up Service Accounts, and providing Integration Secrets, please refer to this **[Documentation](https://docs.consizehq.com/getting-started/installation/#1-create-the-consize-namespace)**.

---

## Features

| Feature | Description |
|---------|-------------|
| **Kubernetes Rightsizing** | Deterministic CPU & Memory p95/p99 usage analysis over 14-day windows. |
| **Cloud Waste Scanning** | Automatically detects unattached EBS volumes, Elastic IPs, and stopped Compute instances (AWS/GCP). |
| **IaC Integration** | Clean up waste directly through the UI or automatically generate PRs against your GitOps repos (Terraform/YAML). |
| **Step-wise Apply** | Large changes are never applied at once; they are broken down into smaller, safe increments. |
| **Auto-Rollback Guardrails** | Monitors SLIs (OOMKills, CPU throttling) after every change. Breaching a threshold triggers an instant, byte-identical rollback. |

---

## Contributing

Consize is built with the community. Look for `good first issue` on the tracker and open a Discussion before larger architectural changes (we log decisions as ADRs). Details in [`CONTRIBUTING.md`](https://github.com/consize-oss/consize/blob/main/CONTRIBUTING.md).

---

## ⚖️ License

Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.

---

## Consize v0.3.0 Is in Development

We are actively building Consize v0.3.0 as the next open-source release. The
work on `main` introduces a safer, extensible foundation for turning
infrastructure evidence into governed optimization actions.

The published v0.2.0 release remains available from the
[`v0.2.0` tag](https://github.com/consize-oss/consize/tree/v0.2.0), with
maintenance work kept on
[`release/0.2`](https://github.com/consize-oss/consize/tree/release/0.2).

### v0.3.0 Features

- **Evidence-backed recommendations:** use real workload metrics to explain what
  should change and why.
- **Policy guardrails:** require approval, allow safe automation or block a
  change based on environment, risk and available evidence.
- **Safety headroom:** retain configurable capacity above observed demand and
  limit the size of each optimization step.
- **Dry runs and reviewable plans:** inspect proposed changes before they affect
  infrastructure.
- **Controlled Kubernetes remediation:** apply approved resource changes through
  one governed action path.
- **Post-action verification:** monitor workload health after every applied
  change.
- **Automatic rollback:** restore the previous configuration when verification
  fails or an action cannot complete safely.
- **Restart recovery:** continue pending actions and verification after Consize
  restarts.
- **Durable audit history:** retain recommendations, policy decisions, actions,
  verification results and rollback outcomes.
- **Extensible plugins:** connect additional infrastructure, metrics, cost and
  action providers through a common plugin model.
- **Clear savings reporting:** keep projected savings separate from savings
  confirmed by provider billing data.

The first `v0.3.0` workflow focuses on Kubernetes Deployments and Prometheus.
Additional cloud providers, databases, billing integrations and GitOps
workflows will follow after the OSS foundation is qualified.
