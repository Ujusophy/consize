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

Helm is the recommended production installation method.

From the Consize repository:

```sh
helm upgrade --install consize ./charts/consize \
  --namespace consize-system \
  --create-namespace \
  -f ./charts/consize/examples/values-prod.yaml
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

## 5. Try Consize

Once Consize is running, you can explore how it works without making changes to a production environment.

Try the [Interactive Sandbox](sandbox.md).

## Next steps

* [Quickstart](quickstart.md)
* [Interactive Sandbox](sandbox.md)
* [Production Installation](installation.md)
* [How Consize Works](../concepts/architecture.md)
* [Configuration](../reference/configuration.md)
