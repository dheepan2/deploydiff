# Releasing DeployDiff

The release workflow builds static archives for macOS and Linux when a version
tag matching `v*` is pushed. Each release contains:

- macOS: `amd64` and `arm64`
- Linux: `amd64` and `arm64`
- `checksums.txt` with SHA-256 checksums

## Create a release

Run the full local verification first:

```bash
go test ./...
go vet ./...
go build ./...
```

Commit and push the intended release source, then create and push a new tag:

```bash
git tag -a v0.1.1 -m "DeployDiff v0.1.1"
git push origin v0.1.1
```

GitHub Actions then creates the GitHub Release and uploads the archives and
checksums. Do not reuse a version tag; publish a new semantic version for each
release.

## Install a release archive

Download the archive matching your operating system and CPU architecture, then:

```bash
tar -xzf deploydiff_v0.1.1_linux_amd64.tar.gz
./deploydiff version
```

Homebrew distribution can use these release archives and their checksums in a
formula after the first automated release is available.
