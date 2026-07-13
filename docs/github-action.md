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

## Pull request example

This workflow compares the `k8s` directory from a pull request's base commit
with its head commit:

```yaml
name: DeployDiff

on:
  pull_request:
    paths:
      - "k8s/**"

permissions:
  contents: read

jobs:
  compare:
    runs-on: ubuntu-latest
    steps:
      - name: Check out base manifests
        uses: actions/checkout@v6
        with:
          ref: ${{ github.event.pull_request.base.sha }}
          path: before

      - name: Check out pull request manifests
        uses: actions/checkout@v6
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          path: after

      - name: Compare deployment impact
        uses: dheepan2/deploydiff@v0.1.2
        with:
          before: before/k8s
          after: after/k8s
          output: table
```

The action creates a PR check and adds the report to the job summary. Posting a
comment directly on a pull request is intentionally not enabled by default.
