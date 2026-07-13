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

## Comparison engine

The comparison engine accepts two parsed deployment states and deterministically
classifies resources as added, removed, or modified. Modified resources include
deterministically ordered field-level changes, including nested map and list
values. Resources are matched by their normalized identity; only their raw
manifest object determines whether a matched resource changed, so moving a
manifest file alone does not create a deployment change. Duplicate identities
are rejected defensively.

The generic comparison model supports Kubernetes resource kinds without a
kind-specific parser. It is explicitly covered for Deployments, Services,
Ingresses, PersistentVolumeClaims, ConfigMaps, and Secrets. Secret `data` and
`stringData` field values are redacted in comparison results.
