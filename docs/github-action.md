# GitHub Action

DeployDiff is packaged as a Docker-based GitHub Action. It compares two
Kubernetes manifest files or directories and writes the rendered report to the
workflow log, job summary, and the `report` action output.

## Direct action inputs

| Input | Required | Description |
| --- | --- | --- |
| `before` | Yes | Path to the base manifest file or directory. |
| `after` | Yes | Path to the changed manifest file or directory. |
| `output` | No | `table` (default), `json`, or `yaml`. |
| `discover-manifests` | No | Discover Kubernetes manifests in arbitrary YAML paths; defaults to `true`. |

## Reusable PR workflow inputs

| Input | Required | Description |
| --- | --- | --- |
| `manifest-path` | No | Repository path to scan for changed manifests; defaults to `.`. |
| `output` | No | `table` (default), `json`, or `yaml`. |
| `comment` | No | Create or update a PR comment when changes exist; defaults to `true`. |

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
  pull-requests: write

jobs:
  compare:
    uses: dheepan2/deploydiff/.github/workflows/compare-pr.yml@v0.1.11
    with:
      manifest-path: . # Or a folder such as deploy/kubernetes
      output: table
      comment: true
```

Use `manifest-path: .` when manifests may live anywhere in the repository. A
more specific path reduces the changed-file scan. Added, deleted, renamed, and
kind-changed manifests are supported because candidates are selected from both
the base and head revisions.

Phase 1 compares changed, plain Kubernetes YAML only. Changes to properties,
Helm values, `Chart.yaml`, Skaffold configuration, and unrendered templates are
not interpreted as deployment changes. Render those inputs before using the
direct action when their deployment impact must be compared.

When `comment` is enabled, the workflow creates one DeployDiff pull request
comment only when deployment changes exist. Later pushes update that comment;
if the deployment diff disappears, the comment is deleted. The caller must
grant `pull-requests: write` as shown above. GitHub gives fork pull requests a
read-only token, so they keep the job-summary report without attempting a
comment.

## Direct action usage

If your workflow already checks out both revisions, use the Docker action
directly with the `before` and `after` paths described above. The action creates
a PR check and adds the report to the job summary. Automatic comments are
provided by the reusable pull request workflow, not the direct Docker action.
