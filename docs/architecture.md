# DeployDiff Architecture

DeployDiff is a deployment intelligence engine for Kubernetes.

The project is built as a layered architecture.

CLI
    ↓
Manifest Loader
    ↓
Parser
    ↓
Knowledge Graph
    ↓
Diff Engine
    ↓
Risk Engine
    ↓
Reporting

Each layer has a single responsibility and is independently testable.

## Manifest loader

The manifest loader is the first M2 component. It reads a YAML file or a
directory tree of `.yaml` and `.yml` files in deterministic path order. It
supports multi-document YAML, ignores empty documents, and validates the
Kubernetes resource identity required by downstream layers: `apiVersion`,
`kind`, and `metadata.name`.

Each loaded resource retains its source path, document number, namespace, and
raw object so parsers and the future diff engine can provide precise results.

## Resource parser and model

The parser normalizes loaded manifests into a resource model with a stable
identity: API group, API version, kind, namespace, and name. It also extracts
labels and annotations, preserves source metadata and the raw object, and
rejects duplicate identities in the same deployment state. This model is the
input to the dependency graph and comparison engine.
