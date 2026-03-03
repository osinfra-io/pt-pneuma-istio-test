# GitHub Copilot Repository Instructions

## Repository Overview

This is a Go application — an example Istio test service that displays GKE cluster information. It is deployed as a container image to Google Artifact Registry and run on GKE clusters managed by pt-pneuma.

- **Language**: Go (see `go.mod` for version)
- **Container**: Multi-stage Docker build (`Dockerfile`), pushed to `us-docker.pkg.dev/pt-corpus-tf16-prod/pt-pneuma-standard/istio-test`
- **Observability**: Datadog APM and tracing via `dd-trace-go`
- **Entry point**: `cmd/http/main.go`
- **Internal packages**: `internal/`

## Code Quality Principles

- **Keep it simple** - Favor straightforward solutions over clever ones.
- **Less is more** - Write only the code necessary to solve the problem at hand.
- **Avoid over-engineering** - Don't add abstraction for hypothetical future needs.
- **Value clarity over brevity** - Explicit, readable code over terse cleverness.
- **Write code for humans first** - Optimize for the next person who reads it.

## GitHub Actions

Two workflows exist in this repo:

- **Sandbox** (`.github/workflows/sandbox.yml`): Runs Go tests on pull requests and manual dispatch.
- **Release** (`.github/workflows/release.yml`): Triggered on published GitHub releases. Runs Go tests, then builds and pushes the container image via WIF to `us-docker.pkg.dev`.

When modifying workflows, update the Mermaid diagram in `README.md` to reflect the changes.

All GitHub Actions must use full 40-character commit hashes instead of version tags:

```yaml
uses: actions/checkout@<40-char-sha> # v<version>
```

## Commit and PR Conventions

- **No Conventional Commits** — do not prefix messages with `feat:`, `fix:`, `chore:`, etc.
- Write commit messages and PR titles in clear, natural language using sentence case.
- Keep titles concise but descriptive.
- PR descriptions should explain what changed, why it changed, and any impact.

✅ **Good:** `Add request timeout to GKE metadata handler`
❌ **Avoid:** `feat: add timeout`

## Branching Workflow

Always `checkout main` and `git pull` before creating a new branch:

```bash
git checkout main && git pull && git checkout -b <branch-name>
```

After a PR is merged, pull `main` and delete the local branch to stay clean.

## Mermaid Diagrams

- Use horizontal layout: `graph LR`
- Always include `color:#000` for text readability on colored backgrounds
- Color palette: `#fff4e6`, `#d4edda`, `#e6d9f5`, `#d1ecf1`, `#fff3cd`, `#f8d7e5`, `#ffdab9`

## References

- [Repository instructions documentation](https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions)

