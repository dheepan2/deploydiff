# Development

## Requirements

- Go 1.23+

## Build

Build a local binary from the repository root:

```bash
go build -o deploydiff .
```

To embed a release version in the binary:

```bash
go build -ldflags "-X github.com/dheepan2/deploydiff/cmd.Version=v0.1.0" -o deploydiff .
```

## Run

```bash
./deploydiff --help
./deploydiff version
```

## Test

```bash
go test ./...
go vet ./...
```

The current unit tests cover the Cobra command tree, version output, global
flag validation, verbose logging, and supported `compare` input forms.

## Verify the CLI

```bash
./deploydiff compare ./before ./after
./deploydiff compare --base origin/main --head HEAD
```

Path-based `compare` runs the manifest loader, resource parser, and comparison
engine, then renders a table by default. Use `--output json` or `--output yaml`
for machine-readable summaries. Git-reference loading is the remaining
integration step.

## Manifest loader

The loader is available as the internal `manifest` package. It accepts a YAML
file or directory and supports multi-document Kubernetes manifests. The
`resource` package parses those manifests into stable Kubernetes identities and
detects duplicate resources. Run both packages' unit tests with the standard
test command above.

## Comparison engine

The internal `diff` package compares parsed deployment states and returns
deterministically ordered added, removed, and modified resources. It is covered
by the standard test command above.

## Functional test fixtures

Reusable end-to-end manifests live in `testdata/compare/before` and
`testdata/compare/after`. The functional command test runs those directories
through the public CLI and verifies table, JSON, and no-change output. You can
also inspect them manually:

```bash
go run . compare ./testdata/compare/before ./testdata/compare/after
```

## Format

```bash
go fmt ./...
```

## Release and deployment

There is no automated release or deployment pipeline yet. For now, distribute
the binary produced by `go build`; CI/CD and release artifacts remain M1 work.

## GitHub Actions CI

The repository includes `.github/workflows/ci.yml`. It runs on pushes, pull
requests, and manual dispatches, and verifies `go test ./...`, `go vet ./...`,
the binary build, fixture comparison, and Docker action image. The workflow
uses read-only repository permissions.
