# Get started

Consize helps engineering teams reduce infrastructure waste without turning cost optimization into a production risk.

It analyzes Kubernetes workloads and cloud resources, recommends safer changes, and lets teams either open a reviewable Infrastructure-as-Code pull request or apply a guarded runtime change.

## Before you start

You need:

* A Kubernetes cluster
* `kubectl` installed and configured
* Helm installed
* Prometheus or a compatible metrics endpoint

Optional integrations such as GitHub, Slack, and cloud provider credentials can be configured later.

## 1. Create the Consize namespace

```sh
kubectl create namespace consize-system
```

## 2. Configure your metrics connection

Create the required secret with your Prometheus endpoint:

```sh
kubectl -n consize-system create secret generic consize-store \
  --from-literal=prometheus-url='http://prometheus-operated.monitoring:9090'
```

> If you use an external Postgres database, configure it separately. Consize can provision a lightweight Postgres database automatically.

## 3. Deploy Consize

Helm is the recommended production installation method. Install directly from the published chart on GitHub Container Registry (GHCR):

```sh
# Export the default values so you can customize your installation
helm show values oci://ghcr.io/consize-oss/charts/consize > values.yaml

# Install using your customized values
helm install consize oci://ghcr.io/consize-oss/charts/consize \
  --version 0.2.0 \
  --namespace consize-system \
  --create-namespace \
  -f values.yaml
```

For more installation options, including cloud credentials, namespace scoping, RBAC, GitHub, and Slack, see [Production Installation](installation.md).

## 4. Verify the installation

Check that the Consize components are running:

```sh
kubectl -n consize-system get pods
```

Then check the scheduled jobs:

```sh
kubectl -n consize-system get cronjobs
```

Finally, check the API health endpoint:

```sh
kubectl -n consize-system port-forward svc/consize-api 18099:8080
```

In another terminal:

```sh
curl http://127.0.0.1:18099/readyz
```

You should get:

```json
{"status":"ready"}
```

## 5. Understand the workflow

Consize follows one basic loop, shown in full (including the rollback path) on the [homepage](../index.md#how-it-works):

```
Observe → Analyze → Recommend → Review → Apply → Verify
```

Consize observes your workloads, analyzes resource usage, and identifies optimization opportunities. Depending on your configuration, the resulting change either goes through a review workflow or is applied directly within configured safety boundaries.

Before enabling automatic changes, review the recommendations Consize produces. For each one, look at:

* Which workload is being optimized
* Which resource is being changed
* The current allocation
* The recommended allocation
* Why the change was recommended

This gives you a chance to understand how Consize behaves before increasing automation.

## 6. Try Consize

Once Consize is running, you can explore how it works without making changes to a production environment.

Try the [Interactive Sandbox](sandbox.md).

## Next steps

* [Interactive Sandbox](sandbox.md)
* [Production Installation](installation.md)
* [How Consize Works](../concepts/architecture.md)
* [Kubernetes Rightsizing](../guides/rightsizing.md)
* [Configuration](../reference/configuration.md)