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
  <a href="https://consizetownhall.slack.com/">
    <img src="https://img.shields.io/badge/Community-Slack-4A154B.svg?logo=slack" alt="Consize community Slack" />
  </a>
</h4>

<img src="/img/demo-dashboard.png" width="100%" alt="Consize dashboard" />

**Consize** is the open source cost optimisation tool that turns cost and usage signals into governed, verifiable optimization actions. It finds waste, explains the safest change, routes it through policy and approval, applies it gradually, verifies health, rolls back on regression, and proves *realized* savings, not just estimated ones.

---

## How is Consize different from Kubecost?

[Kubecost](https://www.kubecost.com/) is a phenomenal open-source project and the absolute gold standard for Kubernetes cost observability and allocation. If your goal is to map cloud billing data to specific namespaces or cross-charge teams, you should use Kubecost.

However, observing waste and **fixing waste safely** are two different problems. 

Consize compliments the ecosystem by focusing entirely on safe, automated action. While observability tools provide recommendations for engineers to apply manually, Consize acts as an active safety net: it generates rightsizing IaC PRs, executes changes in small steps, and monitors your SLIs (e.g., latency, OOM kills) in real-time. If an application degrades, Consize triggers an automated rollback. 

---

## Try the Interactive Sandbox

The fastest way to experience Consize's safety net is through our **Interactive Sandbox**. It runs entirely on your local machine using a single Docker container, pre-seeded with historical data, cloud waste opportunities, and a live metrics simulation.

```bash
# Pull and run the all-in-one interactive sandbox
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

---

## Configuration

For detailed instructions on configuring Helm values, setting up Service Accounts, and providing Integration Secrets, please refer to this **[Installation Guide](docs/customer-guide.md#installation)**.

---

## Core Features

| Feature | Description |
|---------|-------------|
| **Kubernetes Rightsizing** | Deterministic CPU & Memory p95/p99 usage analysis over 14-day windows. |
| **Cloud Waste Scanning** | Automatically detects unattached EBS volumes, Elastic IPs, and stopped Compute instances (AWS/GCP). |
| **IaC Integration** | Clean up waste directly through the UI or automatically generate PRs against your GitOps repos (Terraform/YAML). |
| **Step-wise Apply** | Large changes are never applied at once; they are broken down into smaller, safe increments. |
| **Auto-Rollback Guardrails** | Monitors SLIs (OOMKills, CPU throttling) after every change. Breaching a threshold triggers an instant, byte-identical rollback. |

---

## Documentation

The full docs, including guides, configuration reference, and troubleshooting, live at **[docs.consizehq.com](https://docs.consizehq.com)**. A few starting points:

- **[Vision & The Safety Net](VISION.md)** — why Consize exists and the safety principle it's built around
- **[Get Started](https://docs.consizehq.com/getting-started/get-started/)** — install it and run your first rightsizing change
- **[How Consize Works](https://docs.consizehq.com/concepts/architecture/)** — architecture and data flow
- **[Security & Least Privilege](SECURITY.md)**
- **[Decisions & ADR Log](https://docs.consizehq.com/contributing/decisions/)**

---

## Contributing

Consize is completely open for contribution! We build with the community, not just for the community. 

Whether it's adding support for new cloud providers, fixing bugs, or improving documentation, we welcome all contributions. 

- **Good first issues:** Look for the `good first issue` label on our issue tracker to get started.
- **Design proposals:** For larger architectural changes, please open a GitHub Discussion first. We use Architecture Decision Records (ADRs) to document significant decisions.
- **Local Development:** Check out the [Architecture & Data Flow](docs/architecture.md) documentation to understand how the components fit together before spinning up your local environment.

---

## ⚖️ License

Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.
