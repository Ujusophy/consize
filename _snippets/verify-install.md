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

If any of these come back unhealthy, see [Troubleshooting](../resources/troubleshooting.md#installation).
