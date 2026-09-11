# Quickstart

This guide gets you from a fresh Consize installation to your first working check.

If you have not installed Consize yet, start with [Get started](get-started.md).

## 1. Check that Consize is running

First, check the Consize pods:

```sh
kubectl -n consize-system get pods
```

You should see the Consize components running.

Then check the scheduled jobs:

```sh
kubectl -n consize-system get cronjobs
```

## 2. Check the API

Forward the Consize API to your local machine:

```sh
kubectl -n consize-system port-forward svc/consize-api 18099:8080
```

Keep this terminal running.

In another terminal, run:

```sh
curl http://127.0.0.1:18099/readyz
```

You should get:

```json
{"status":"ready"}
```

If you get this response, the Consize API is ready.

## 3. Understand the workflow

Consize follows this basic workflow:

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
```

Consize first observes your workloads, analyzes their resource usage, and identifies potential optimization opportunities.

Depending on your configuration, the resulting change can either go through a review workflow or be applied directly within configured safety boundaries.

## 4. Start with recommendations

Before enabling automatic changes, start by reviewing the recommendations produced by Consize.

Look at:

* Which workload is being optimized
* Which resource is being changed
* The current allocation
* The recommended allocation
* Why the change was recommended

This gives you a chance to understand how Consize behaves before increasing automation.

## 5. Try the Interactive Sandbox

Want to explore Consize without changing a real environment?

Try the [Interactive Sandbox](sandbox.md).

## Next steps

* [Interactive Sandbox](sandbox.md)
* [Production Installation](installation.md)
* [Kubernetes Rightsizing](../guides/rightsizing.md)
* [The Safety Net](../concepts/safety-net.md)
* [Configuration](../reference/configuration.md)
