# Contributing

Contributions are welcome at
[github.com/ankurs/kb](https://github.com/ankurs/kb).

## Layout

| Path | Contents |
| --- | --- |
| `cmd/kb` | CLI entry point. |
| `internal/cli` | Command definitions. |
| `internal/config` | Loading and validating `kb.yaml`. |
| `internal/workspace` | Markdown discovery, links, navigation, capture, search. |
| `internal/render` | Serving, building, publishing; browser code in `web/`. |
| `assets/browser` | Pinned third-party browser assets and their licenses. |
| `testdata/sample` | The example workspace, also a test fixture. |
| `site` | This documentation, a `kb` workspace. |

## Build and test

```sh
go build -o /tmp/kb ./cmd/kb
go test ./...
go test -race -cover ./...      # what CI runs
go vet ./...
gofmt -l $(git ls-files '*.go')
go run ./cmd/kb --workspace testdata/sample check
go run ./cmd/kb --workspace site check
```

Browser rendering tests use headless Chromium when available. Use a
disposable copy of `testdata/sample` for `build`, `verify`, and `publish`
smoke tests.

## Guidelines

- Add focused regression tests when changing link resolution, mounts,
  security checks, publishing selection, or output staleness.
- Keep browser assets offline. When vendoring a dependency, update
  `assets/browser/manifest.json` and the license notices.
- Pull requests should explain behavior and risk, mention the tests run, and
  include screenshots for visible viewer changes.
- Update this site in the same pull request as user-visible behavior changes.
