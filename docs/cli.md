# CLI

## deploydiff

Root command.

Global Flags

--config
--verbose
--output

Commands

compare
version
completion

## compare

Compare two manifest directories:

```bash
deploydiff compare ./before ./after
```

Compare the current branch with a Git reference without creating checkout
directories yourself:

```bash
deploydiff compare --base origin/main --head HEAD
```

When your repository contains non-production fixture manifests, target only the
manifest directory:

```bash
deploydiff compare --base origin/main --head HEAD --path deploy/kubernetes
```

Path-based comparison loads Kubernetes YAML from each file or directory and
reports added, removed, and modified resources. Modified resources include
field-level values, for example `spec.replicas: 2 → 3`. Git-reference comparison
uses `git archive` to materialize both revisions in temporary directories, which
are removed automatically. It discovers Kubernetes manifests recursively and
ignores unrelated valid YAML by default. `--path` defaults to `.`; a path that
does not exist in one revision is treated as an empty deployment state, so
adding or removing a manifest directory is reported correctly.

Deployments, Services, Ingresses, PersistentVolumeClaims, ConfigMaps, and
Secrets are all supported by the generic manifest comparison. Secret `data` and
`stringData` values are always rendered as `<redacted>`.

Use the global output flag to select a report format:

```bash
deploydiff --output table compare ./before ./after
deploydiff --output json compare ./before ./after
deploydiff --output yaml compare ./before ./after
```

Use `--discover` when the input directory includes unrelated YAML such as Helm
values, GitHub workflow files, or application configuration. DeployDiff then
loads only documents with Kubernetes identity fields:

```bash
deploydiff compare --discover ./before-repository ./after-repository
```

## version

Print the version embedded in the binary at build time.
