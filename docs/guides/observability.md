# Observability

Consize uses observation and verification to make infrastructure optimization safer.

The basic idea is simple:

```text
Observe → Change → Verify
```

Consize needs visibility into workload behavior before it can make useful optimization recommendations.

## What Consize observes

Depending on your configuration, Consize can use information such as:

* CPU usage
* Memory usage
* Resource requests
* Resource limits
* Kubernetes workload state
* Historical resource usage

This information helps Consize understand how resources are being used before recommending a change.

## Before a change

Before applying an optimization, Consize evaluates the available workload data.

For example, if a workload has a high CPU request but consistently uses much less CPU, Consize can identify that as a potential rightsizing opportunity.

The recommendation is then evaluated against the configured safety boundaries.

## After a change

Verification is an important part of the optimization workflow. After a change is applied, Consize compares health signals before and after, on the changed workload specifically, not the whole namespace.

By default, it checks:

* Restarts
* OOM kills
* Evictions
* CPU throttling
* Optional application-level latency or error-rate metrics, if you've configured them

Verification returns one of three verdicts:

| Verdict | Meaning |
|---|---|
| `passed` | The change looks safe. The saving is counted as realized. |
| `failed` | The change likely caused a regression. Consize rolls back and records evidence. |
| `inconclusive` | Not enough data to prove safety. No automatic rollback, but it requires human attention. |

The default verification window starts at **1 hour and scales with the apply step**, step 1 verifies after 1 hour, step 2 after 2 hours, step 3 after 3 hours, and so on. This keeps the first feedback loop fast while giving deeper reductions more observation time. The verifier itself runs every minute, picking up applies automatically as soon as their window opens, it never verifies early.

## Health checks

If you just completed [Get started](../getting-started/get-started.md#4-verify-the-installation) or [Production Installation](../getting-started/installation.md#7-verify-the-installation), you've already confirmed the API is healthy and the Consize pods and cronjobs are running. Those pages have the exact commands if you need to re-check.

## Verification is not monitoring

Consize's verification workflow does not replace your existing application monitoring.

Your existing monitoring tools should remain responsible for application-level signals such as:

* Request latency
* Error rates
* Availability
* Application health
* Business metrics

Consize complements these systems by focusing on infrastructure optimization and the effects of resource changes.

## The feedback loop

This is the full loop described on the [homepage](../index.md#how-it-works), it doesn't stop at Verify:

```
Observe → Analyze → Recommend → Review → Apply → Verify → Observe again
```

This feedback loop helps teams optimize infrastructure without treating resource changes as one-time actions.

## Next steps

* [Kubernetes Rightsizing](rightsizing.md)
* [The Safety Net](../concepts/safety-net.md)
* [How Consize Works](../concepts/architecture.md)
* [Production Installation](../getting-started/installation.md)