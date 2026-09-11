# Kubernetes Rightsizing

Kubernetes workloads are often given more CPU and memory than they actually need.

Rightsizing helps reduce that waste by adjusting resource requests and limits based on observed workload behavior.

Consize helps identify these opportunities and turn them into safer, actionable changes.

## What Consize looks at

Consize analyzes workload resource usage, including:

* CPU usage
* Memory usage
* CPU requests
* Memory requests
* CPU limits
* Memory limits
* Workload configuration
* Historical usage patterns

The goal is to compare **what a workload has been allocated** with **what it actually uses**.

## A simple example

Imagine a deployment has:

```yaml
resources:
  requests:
    cpu: "1000m"
    memory: "2Gi"
```

But the workload consistently uses much less than that.

Consize can identify the difference as a potential optimization opportunity.

Instead of immediately changing the workload, the recommendation can go through the configured review and safety process.

## The rightsizing workflow

```text
Collect usage
     ↓
Analyze workload
     ↓
Identify opportunity
     ↓
Generate recommendation
     ↓
Safety checks
     ↓
Review or apply
     ↓
Verify
```

## Recommendations

A recommendation should answer three questions:

### What should change?

For example:

```text
CPU request: 1000m → 500m
Memory request: 2Gi → 1Gi
```

### Why should it change?

The recommendation is based on observed resource usage and the configured optimization strategy.

### Is the change safe?

Consize evaluates the proposed change against its configured safety boundaries before it can be applied automatically.

## Review before applying

You can keep rightsizing inside your existing Infrastructure-as-Code workflow.

A typical workflow is:

```text
Consize
   ↓
Recommendation
   ↓
Infrastructure-as-Code change
   ↓
Pull request
   ↓
Review
   ↓
Merge
   ↓
Deploy
```

This is useful for teams that want optimization recommendations without giving an automated system direct write access to production.

## Direct runtime changes

Teams that enable direct application can allow Consize to update supported workloads directly.

Write access should be limited to the namespaces and resources where runtime optimization is allowed.

See [The Safety Net](../concepts/safety-net.md) for more information about permissions and guardrails.

## Start with a small scope

You do not need to optimize every workload at once.

Start with:

1. A small number of namespaces
2. A few representative workloads
3. Recommendations before automatic application
4. A review of the resulting changes
5. Gradual expansion once the results are understood

This makes it easier to validate the optimization process before increasing automation.

## Monitoring after a change

Rightsizing should not end when the resource configuration changes.

After a change, monitor the workload for:

* Increased CPU pressure
* Increased memory pressure
* Restarts
* Failed deployments
* Increased latency
* Other application-specific signals

Consize's verification workflow helps close this loop.

## Next steps

* [The Safety Net](../concepts/safety-net.md)
* [How Consize Works](../concepts/architecture.md)
* [Production Installation](../getting-started/installation.md)
* [Configuration](../reference/configuration.md)
