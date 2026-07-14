# GitHub Action

DeployDiff is packaged as a Docker-based GitHub Action. It compares two
Kubernetes manifest files or directories and writes the rendered report to the
workflow log, job summary, and the `report` action output.

## Inputs

| Input | Required | Description |
| --- | --- | --- |
| `before` | Yes | Path to the base manifest file or directory. |
| `after` | Yes | Path to the changed manifest file or directory. |
| `output` | No | `table` (default), `json`, or `yaml`. |
| `discover-manifests` | No | Discover Kubernetes manifests in arbitrary YAML paths; defaults to `true`. |

## Pull request workflow (recommended)

Call DeployDiff's reusable workflow from a workflow in the repository you want
to protect. It checks out the pull request's base and head commits, selects
changed YAML files with a top-level `kind:` in either revision, and adds the
comparison report to the job summary.

```yaml
name: DeployDiff

on:
  pull_request:
    paths:
      - "**/*.yaml"
      - "**/*.yml"

permissions:
  contents: read

jobs:
  compare:
    uses: dheepan2/deploydiff/.github/workflows/compare-pr.yml@v0.1.10
    with:
      manifest-path: . # Or a folder such as deploy/kubernetes
      output: table
```

Use `manifest-path: .` when manifests may live anywhere in the repository. A
more specific path reduces the changed-file scan. Added, deleted, renamed, and
kind-changed manifests are supported because candidates are selected from both
the base and head revisions.

Phase 1 compares changed, plain Kubernetes YAML only. Changes to properties,
Helm values, `Chart.yaml`, Skaffold configuration, and unrendered templates are
not interpreted as deployment changes. Render those inputs before using the
direct action when their deployment impact must be compared.

## Direct action usage

If your workflow already checks out both revisions, use the Docker action
directly with the `before` and `after` paths described above. The action creates
a PR check and adds the report to the job summary. Posting a comment directly
on a pull request is intentionally not enabled by default.
