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

## Pull request example

This workflow compares all Kubernetes manifests found recursively in a pull
request's base and head commits, while ignoring unrelated YAML files:

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
        uses: dheepan2/deploydiff@v0.1.3
        with:
          before: before
          after: after
          output: table
```

The action creates a PR check and adds the report to the job summary. Posting a
comment directly on a pull request is intentionally not enabled by default.
