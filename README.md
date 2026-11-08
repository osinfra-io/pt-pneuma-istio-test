# Istio Test

[![Dependabot](https://img.shields.io/github/actions/workflow/status/osinfra-io/pt-pneuma-istio-test/dependabot.yml?style=for-the-badge&logo=github&color=2088FF&label=Dependabot)](https://github.com/osinfra-io/pt-pneuma-istio-test/actions/workflows/dependabot.yml)

A small validation application for Pneuma-managed Istio gateways. It exposes health, GKE metadata, and authenticated identity endpoints used to verify routing and gateway authentication.

`pt-pneuma-istio-test` builds and publishes the image; `pt-pneuma` owns its Kubernetes deployment under `regional/istio/test`.

## GitHub Actions Workflows

**Workflow Details:**

- **Sandbox**: Triggered on pull request (opened, synchronize), excluding .md files; manual dispatch — runs Go tests, validates the OpenAPI specification, and uploads coverage reports to Datadog
- **Release**: Triggered on a published GitHub release — runs Go tests, then uses the shared Techne workflow to publish versioned and `latest` tags to `us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test`
- **Registry**: `us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test`
- **Authentication**: Workload Identity Federation via `pt-pneuma-github@pt-corpus-tf16-prod.iam.gserviceaccount.com`

### Datadog Test Optimization

The Sandbox and Release Go test jobs use Datadog Test Optimization auto-instrumentation. Test results are sent to `datadoghq.com` with service `istio-test` and environment `ci`; Sandbox also uploads its existing coverage report.

Both jobs reuse the `DATADOG_API_KEY` GitHub Actions secret. It must contain an API key from the destination Datadog organization, managed in [API key settings](https://app.datadoghq.com/organization-settings/api-keys) (see the [API key documentation](https://docs.datadoghq.com/account_management/api-app-keys/)). Store the key in GitHub Actions secrets, never in source files. When the secret is unavailable, including fork and Dependabot pull requests, Test Optimization setup is skipped and the Go tests still run normally. The separate coverage upload retains its existing behavior.

After running either workflow with the secret available, confirm the expected tests and outcomes for that commit in [Test Optimization Explorer](https://app.datadoghq.com/ci/test/runs?query=@git.repository.name:osinfra-io/pt-pneuma-istio-test). A passing CI job alone does not confirm that Datadog received the test results.

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
- `GET /istio-test/auth` — protected authentik identity details. The gateway injects trusted identity headers after sign-in; the application does not expose the JWT or session cookie.

The [OpenAPI specification](openapi.yaml) describes these endpoints and is referenced by the `istio-test` API entity in Datadog Catalog. The spec is not served by the application.

A successful authenticated `/auth` response is JSON similar to:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "name": "Alice Example",
  "uid": "user-123",
  "groups": ["all", "platform"],
  "authentik": {
    "application": "pneuma-sb-osinfra-io",
    "googleEmail": "alice@example.com",
    "outpost": "authentik Embedded Outpost",
    "provider": "pneuma.sb.osinfra.io",
    "version": "2025.x"
  },
  "request": {
    "authentikHeaders": ["X-Authentik-Email", "X-Authentik-Groups", "X-Authentik-Name", "X-Authentik-Uid", "X-Authentik-Username"],
    "forwardedFor": "203.0.113.10",
    "forwardedHost": "pneuma.sb.osinfra.io",
    "forwardedProto": "https",
    "host": "pneuma.sb.osinfra.io",
    "peerIdentity": "spiffe://cluster.local/ns/istio-system/sa/gateway",
    "requestId": "0f6c1c3e-...",
    "traceparent": "00-..."
  },
  "workload": {
    "namespace": "istio-test",
    "pod": "istio-test-6d8f7c9b5-abcde",
    "version": "v1.2.3"
  }
}
```

The `request` and `workload` objects help troubleshoot routing, authentication and the mesh. Tokens, cookies and the `Authorization` header are never returned, and only the names of the `X-Authentik-*` headers received are listed. Cluster and zone details stay on the public `/metadata/*` endpoints.

For local development, port-forward the service and query a public endpoint:

```bash
curl http://localhost:8080/istio-test/health
```
