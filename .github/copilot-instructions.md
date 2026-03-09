# pt-pneuma-istio-test

A Go application — an example Istio test service that displays GKE cluster information. Deployed as a container image to Google Artifact Registry and run on GKE clusters managed by pt-pneuma.

- **Language**: Go (see `go.mod` for version)
- **Container**: Multi-stage Docker build (`Dockerfile`), pushed to Google Artifact Registry (see `.github/workflows/` for registry path)
- **Observability**: Datadog APM and tracing via `dd-trace-go`
- **Entry point**: `cmd/http/main.go`
- **Internal packages**: `internal/`
