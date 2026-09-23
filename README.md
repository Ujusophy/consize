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
| Run everything locally in one container, pre-seeded with data and a live regression to watch Consize catch and roll back. | Install onto a live Kubernetes cluster on AWS/GCP with the official Helm chart. Read-only by default until you grant least-privilege `RoleBindings`. |

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

## Roadmap
**Full scope & progress:** see [`ROADMAP.md`](ROADMAP.md) for phase-by-phase detail and how to weigh in.

---

## Contributing

Consize is built with the community. Look for `good first issue` on the tracker and open a Discussion before larger architectural changes (we log decisions as ADRs). Details in [`CONTRIBUTING.md`](https://github.com/consize-oss/consize/blob/main/CONTRIBUTING.md).

---

## ⚖️ License

Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.
