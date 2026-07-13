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

`compare` currently validates these invocation forms. The manifest loader is
available, while resource parsing and the comparison engine are the next M2
steps.

## Manifest loader

The loader is available as the internal `manifest` package. It accepts a YAML
file or directory and supports multi-document Kubernetes manifests. The
`resource` package parses those manifests into stable Kubernetes identities and
detects duplicate resources. Run both packages' unit tests with the standard
test command above.

## Format

```bash
go fmt ./...
```

## Release and deployment

There is no automated release or deployment pipeline yet. For now, distribute
the binary produced by `go build`; CI/CD and release artifacts remain M1 work.
