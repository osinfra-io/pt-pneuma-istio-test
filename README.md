# Istio Test

[![Dependabot](https://img.shields.io/github/actions/workflow/status/osinfra-io/pt-pneuma-istio-test/dependabot.yml?style=for-the-badge&logo=github&color=2088FF&label=Dependabot)](https://github.com/osinfra-io/pt-pneuma-istio-test/actions/workflows/dependabot.yml)

A small validation application for Pneuma-managed Istio gateways. It exposes health, GKE metadata, and authenticated identity endpoints used to verify routing and gateway authentication.

`pt-pneuma-istio-test` builds and publishes the image; `pt-pneuma` owns its Kubernetes deployment under `regional/istio/test`.

## GitHub Actions Workflows

**Workflow Details:**

- **Sandbox**: Triggered on pull request (opened, synchronize), excluding .md files; manual dispatch — runs Go tests and uploads coverage reports to Datadog
- **Release**: Triggered on a published GitHub release — runs Go tests, then uses the shared Techne workflow to publish versioned and `latest` tags to `us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test`
- **Registry**: `us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test`
- **Authentication**: Workload Identity Federation via `pt-pneuma-github@pt-corpus-tf16-prod.iam.gserviceaccount.com`

## Usage

```yaml
---
apiVersion: v1
kind: Namespace

metadata:
  name: istio-test

---
apiVersion: apps/v1
kind: Deployment

metadata:
  name: istio-test
  namespace: istio-test

spec:
  replicas: 1
  selector:
    matchLabels:
      app: istio-test

  template:
    metadata:
      labels:
        app: istio-test

    spec:
      containers:
        - image: us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test:latest
          imagePullPolicy: Always
          name: istio-test

          ports:
            - containerPort: 8080

          resources:
            limits:
              cpu: "50m"
              memory: "128Mi"
            requests:
              cpu: "25m"
              memory: "64Mi"

---
apiVersion: v1
kind: Service

metadata:
  name: istio-test
  namespace: istio-test

  labels:
    app: istio-test

spec:
  ports:
    - name: http
      port: 8080
      targetPort: 8080

  selector:
    app: istio-test

```

After deploying, you can get the information about the GKE cluster by running the following command:

```bash
kubectl port-forward --namespace istio-test $(kubectl get pod --namespace istio-test --selector="app=istio-test" --output jsonpath='{.items[0].metadata.name}') 8080:8080
```

Available endpoints:

- `GET /istio-test/health` — public enhanced health check.
- `GET /istio-test/metadata/{cluster-name|cluster-location|instance-zone}` — public GKE metadata details.
- `GET /istio-test/auth` — protected Authentik identity details. The gateway injects trusted identity headers after sign-in; the application does not expose the JWT or session cookie.

A successful authenticated `/auth` response is JSON similar to:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "name": "Alice Example",
  "uid": "user-123",
  "groups": ["all", "platform"]
}
```

For local development, port-forward the service and query a public endpoint:

```bash
curl http://localhost:8080/istio-test/health
```
