# Install

`kb` is a single binary with no runtime dependencies. Browser assets are
embedded, so the binary is all you need.

## With Go (recommended)

Requires Go 1.24 or later.

```sh
go install github.com/ankurs/kb/cmd/kb@latest
kb --help
```

The binary is installed to `$(go env GOPATH)/bin`. If `kb` is not found, add
that directory to your `PATH`, for example in your shell profile:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Run the same `go install` command again to upgrade. To pin a version, replace
`@latest` with a release tag such as `@v0.1.2`.

**Older Go-installed versions:** If the viewer loads `_kb` scripts but returns
404 for `/vendor/` JavaScript and CSS, its Go module archive omitted the
embedded browser assets. This affects v0.1.2; reinstalling that same tag
will not fix it. Until a version containing the packaging fix is released,
use a release archive or build from a source checkout. Then run the Go
install command again to upgrade.

## From a release archive

Each [GitHub release](https://github.com/ankurs/kb/releases) includes archives
for macOS and Linux on `amd64` and `arm64`, plus a `checksums.txt` file. Use
this when Go is not installed.

```sh
version=v0.1.2          # pick the latest from the releases page
os=darwin               # or linux
arch=arm64              # or amd64
curl -LO "https://github.com/ankurs/kb/releases/download/$version/kb_${version}_${os}_${arch}.tar.gz"
curl -LO "https://github.com/ankurs/kb/releases/download/$version/checksums.txt"
shasum -a 256 --ignore-missing -c checksums.txt
tar -xzf "kb_${version}_${os}_${arch}.tar.gz"
sudo mv kb /usr/local/bin/
```

On Linux, `sha256sum --ignore-missing -c checksums.txt` performs the same
check.

## From source

```sh
git clone https://github.com/ankurs/kb.git
cd kb
go build -o kb ./cmd/kb
```

`go run ./cmd/kb` also works from the source directory without installing.
See [Contributing](../contributing.md) for tests and checks.

## Optional: semantic search

`kb find` and every reading and publishing command work with no extra
software. For hybrid semantic retrieval, install
[qmd](https://github.com/tobi/qmd) separately; see [Search](../guides/search.md).

## Verify the install

```sh
kb doctor
```

Run inside a workspace, `kb doctor` prints the workspace root, the config file
in use, and any optional qmd setup that is missing. Next, follow the
[quick start](README.md).
