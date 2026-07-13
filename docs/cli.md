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

Or compare two Git references:

```bash
deploydiff compare --base origin/main --head HEAD
```

The command validates its input today; manifest comparison will be added with the
manifest loader and diff engine.

## version

Print the version embedded in the binary at build time.
