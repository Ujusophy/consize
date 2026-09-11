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

Verification is an important part of the optimization workflow.

After a change is applied, the workload should continue to be monitored for unexpected behavior.

Useful signals include:

* CPU pressure
* Memory pressure
* Restarts
* Failed deployments
* Increased latency
* Application-specific health signals

## Health checks

You can check whether the Consize API is ready with:

```sh id="9iz3wr"
kubectl -n consize-system port-forward svc/consize-api 18099:8080
```

Then, from another terminal:

```sh id="2l1f9m"
curl http://127.0.0.1:18099/readyz
```

A healthy API should return:

```json id="7h3d2x"
{"status":"ready"}
```

## Check Consize workloads

Check the Consize pods:

```sh id="6k7m8p"
kubectl -n consize-system get pods
```

You can also check scheduled jobs:

```sh id="8n4q1s"
kubectl -n consize-system get cronjobs
```

These commands help confirm that the main Consize components are running.

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

The complete workflow is:

```text
Observe
   ↓
Analyze
   ↓
Recommend
   ↓
Safety checks
   ↓
Apply
   ↓
Verify
   ↓
Observe again
```

This feedback loop helps teams optimize infrastructure without treating resource changes as one-time actions.

## Next steps

* [Kubernetes Rightsizing](rightsizing.md)
* [The Safety Net](../concepts/safety-net.md)
* [How Consize Works](../concepts/architecture.md)
* [Production Installation](../getting-started/installation.md)
