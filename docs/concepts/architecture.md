# How Consize Works

Consize continuously observes your infrastructure, identifies optimization opportunities, evaluates the safety of proposed changes, and verifies the result after changes are applied.

At a high level:

```text
Observe → Analyze → Recommend → Review → Apply → Verify
```

## 1. Observe

Consize collects information about your infrastructure and workload behavior.

This includes things such as:

* Kubernetes resources
* Resource requests and limits
* CPU and memory usage
* Workload configuration
* Infrastructure metadata
* Cloud resource information

The goal is to understand what is actually happening before recommending a change.

## 2. Analyze

Consize compares resource usage with the resources currently allocated.

For example, a workload may consistently use significantly less CPU and memory than it has been allocated.

Consize can identify this as a potential rightsizing opportunity.

## 3. Recommend

Consize turns identified opportunities into proposed changes.

Recommendations are based on observed usage and configured safety boundaries rather than simply applying the most aggressive possible reduction.

## 4. Review

Changes can go through a review workflow before they are applied.

For Infrastructure-as-Code workflows, Consize can produce changes that can be reviewed through your existing pull request process.

This gives engineering teams an opportunity to inspect and approve a change before it reaches production.

## 5. Apply

Teams can choose how changes are applied.

### Infrastructure-as-Code

Changes can be reviewed and merged through your existing Infrastructure-as-Code workflow.

### Runtime changes

For teams that enable direct application, Consize can apply approved changes directly to configured resources.

Runtime changes are restricted by Kubernetes permissions and configured safety boundaries.

## 6. Verify

Applying a change is not the end of the process.

Consize verifies the resulting state and checks whether the workload continues to behave as expected.

This creates a feedback loop:

```text
Observe
   ↓
Analyze
   ↓
Recommend
   ↓
Review
   ↓
Apply
   ↓
Verify
   ↓
Observe again
```

## The safety layer

The important part of this workflow is that optimization does not happen in isolation.

Consize evaluates proposed changes against safety rules before they are applied.

This is what allows teams to automate optimization while keeping control over what can change and where.

Learn more in [The Safety Net](safety-net.md).

## Deployment model

Consize runs inside your Kubernetes environment and uses Kubernetes-native permissions to control access.

A typical deployment includes:

* A collector that gathers infrastructure and workload information
* An analysis and recommendation layer
* An API for interacting with Consize
* Scheduled jobs for recurring analysis
* Kubernetes RBAC for controlling access
* Optional integrations for notifications and external workflows

For installation details, see [Production Installation](../getting-started/installation.md).

## Next steps

* [The Safety Net](safety-net.md)
* [Production Installation](../getting-started/installation.md)
* [Configuration](../reference/configuration.md)
* [Quickstart](../getting-started/quickstart.md)
