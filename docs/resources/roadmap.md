# Roadmap

Consize is being developed as an open-source infrastructure optimization platform.

The roadmap focuses on making optimization safer, easier to adopt, and useful across more infrastructure environments.

## Available today

Some capabilities are already shipped, not just planned:

* **Kubernetes rightsizing**, percentile-based CPU/memory recommendations, see [Kubernetes Rightsizing](../guides/rightsizing.md)
* **Cloud database rightsizing** for AWS RDS and GCP Cloud SQL
* **Cloud waste scanning**
* **The safety engine**, guardrails, step-wise apply, and automatic rollback on regression, see [The Safety Net](../concepts/safety-net.md)
* **Reviewable Infrastructure-as-Code changes** through your existing pull request workflow, as an alternative to direct runtime application
* **Slack and GitHub integrations**

## Current focus

### Kubernetes rightsizing

Improve how Consize identifies and recommends resource optimizations for Kubernetes workloads.

* CPU rightsizing
* Memory rightsizing
* Better usage analysis
* Safer recommendations
* Improved verification

### Safety and automation

Make automated infrastructure changes more controlled and predictable.

* Stronger safety checks
* More granular permissions
* Better verification
* Controlled automatic application
* Improved failure handling

### Developer experience

Make Consize easier to install, understand, and operate.

* Clearer documentation
* Better configuration defaults
* Improved CLI and API experience
* Easier local development
* Better debugging and observability

## Future areas

As the project develops, the roadmap may expand into areas such as:

* Support for additional cloud providers
* Additional Infrastructure-as-Code workflow integrations
* Additional notification and collaboration integrations
* More optimization strategies

These areas are exploratory and may change as the project evolves.

## How to contribute

Consize is open source, and contributions are welcome.

If you want to work on a roadmap item, start by:

1. Checking the existing issues and discussions.
2. Understanding the current architecture.
3. Opening or joining a discussion about the proposed change.
4. Testing the change before submitting a pull request.

See the [Contributing](../contributing/testing.md) documentation for development and testing information.

## Roadmap changes

The roadmap is not a fixed commitment.

Priorities can change based on:

* User feedback
* Production experience
* Community contributions
* Technical constraints
* Project priorities

The goal is to build the most useful and safest version of Consize over time.