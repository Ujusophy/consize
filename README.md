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
  <a href="https://github.com/consize-oss/consize/blob/main/docs/architecture.md">Architecture</a> |
  <a href="https://github.com/consize-oss/consize/blob/main/SECURITY.md">Security</a>
</h4>

<h4 align="center">
  <a href="https://github.com/consize-oss/consize/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Consize is released under the Apache 2.0 license." />
  <a href="https://consizetownhall.slack.com/">
    <img src="https://img.shields.io/badge/Community-Slack-4A154B.svg?logo=slack" alt="Consize community Slack" />
  </a>
</h4>

<img src="/img/demo-dashboard.png" width="100%" alt="Consize dashboard" />

**Consize** is the open source cost optimisation tool that turns cost and usage signals into governed, verifiable optimization actions. It finds waste, explains the safest change, routes it through policy and approval, applies it gradually, verifies health, rolls back on regression, and proves *realized* savings, not just estimated ones.

---

## Features

**Find waste**
- **Resource Rightsizing**: deterministic CPU & memory p95/p99 usage analysis over rolling windows, with support for more resource types landing through the plugin system.
- **Cloud Waste Scanning**: detects unattached volumes, idle IPs, and stopped compute instances (AWS/GCP).

**Act safely**
- **Policy Engine**: every recommendation carries a decision: `blocked`, `approval_required`, `open_pr`, or `auto_apply`.
- **Step-wise Apply**: large changes are never applied at once; they're broken into small, reversible increments.
- **IaC Integration**: clean up waste via the UI, or auto-generate PRs against your GitOps repos (Terraform/YAML).

**Verify and recover**
- **Auto-Rollback Guardrails**: watches SLIs (OOMKills, CPU throttling, latency, error rate) after every change and triggers an instant, byte-identical rollback on regression.
- **Audit Trail**: every recommendation, approval, and action is recorded, so realized savings can be traced back to evidence.

**Extend it**
- **Plugin SDK** *(0.3.0-alpha)*: a versioned contract for building new metrics, cost, and action providers without touching Consize's core engine.

---

## Getting Started

| 🧪 Try the Sandbox | 🚀 Production Install |
| --- | --- |
| Run everything locally in one container, pre-seeded with data and a live regression to watch Consize catch and roll back. | Install onto a live cluster (AWS/GCP) with the official Helm chart. Read-only by default until you grant least-privilege `RoleBindings`. |

### Try the Interactive Sandbox

```bash
docker run -p 3000:3000 -p 8080:8080 -it ghcr.io/consize-oss/consize-sandbox:latest
```

Open `http://localhost:3000` and watch the Verifier catch an intentional regression on the `checkout-api` workload, then automatically roll it back.

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

Full setup (Service Accounts, Integration Secrets, Helm values) is in the **[Documentation](https://docs.consizehq.com/getting-started/installation/#1-create-the-consize-namespace)**.

---

## Contributing

Consize is built with the community. Look for `good first issue` on the tracker and open a Discussion before larger architectural changes (we log decisions as ADRs). Details in [`CONTRIBUTING.md`](https://github.com/consize-oss/consize/blob/main/CONTRIBUTING.md).

---

## ⚖️ License

Distributed under the **Apache 2.0 License**. See [`LICENSE`](https://github.com/consize-oss/consize/blob/main/LICENSE).
