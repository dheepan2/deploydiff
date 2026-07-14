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

## Phase 1 pull request scope

The reusable GitHub workflow is the Phase 1 product boundary. It obtains the
changed paths between the pull request base and head revisions and selects
`.yaml` and `.yml` files containing a top-level `kind:` in either revision.
Both versions of each candidate are compared, so additions, deletions, renames,
and resource-kind changes are represented correctly.

Phase 1 does not infer deployment impact from properties, Helm values,
`Chart.yaml`, Skaffold configuration, or unrendered templates. Those inputs need
an explicit render stage before the direct action. Native render orchestration
and a stable end-user CLI contract remain future design work.

The reusable workflow owns PR-comment delivery. It upserts a single comment
identified by a hidden marker when a deployment diff exists and removes stale
comments when a later push has no deployment changes. Comment delivery is
best-effort and never replaces the job summary, which remains available for
fork pull requests and repositories without a writable pull-request token.
