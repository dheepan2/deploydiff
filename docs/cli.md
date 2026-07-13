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

Git-reference comparison is planned:

```bash
deploydiff compare --base origin/main --head HEAD
```

Path-based comparison loads Kubernetes YAML from each file or directory and
reports added, removed, and modified resources. Git-reference comparison will
be enabled once the Git manifest reader is implemented.

Use the global output flag to select a report format:

```bash
deploydiff --output table compare ./before ./after
deploydiff --output json compare ./before ./after
deploydiff --output yaml compare ./before ./after
```

## version

Print the version embedded in the binary at build time.
