# Repository Guidelines

## Project Structure & Module Organization

`cmd/kb` is the CLI entry point. `internal/cli` defines commands; `internal/config` loads and validates `kb.yaml`; `internal/workspace` discovers Markdown, links, and navigation; `internal/render` serves and builds the viewers. Browser scripts and styles live in `internal/render/web`, while pinned third-party assets and licenses live in `assets/vendor`. The fictional workspace in `testdata/sample` is both documentation and a test fixture. Keep generated `.kb/` and `dist/` output out of commits unless explicitly requested.

Keep the root `README.md` and the user documentation in `site/` in sync with behavior, commands, configuration, examples, and security changes. Update both in the same change rather than leaving either outdated.

## Build, Test, and Development Commands

- `go build -o /tmp/kb ./cmd/kb` builds the CLI.
- `go test ./...` runs unit and integration tests; `go test -race -cover ./...` matches CI's stronger check.
- `go vet ./...` and `gofmt -l $(git ls-files '*.go')` check analysis and formatting.
- `go run ./cmd/kb --workspace testdata/sample check` validates the example content. For `build`, `verify`, or `publish` smoke tests, use a disposable copy of `testdata/sample` so the fixture stays unchanged.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`), tabs as produced by the formatter, short package names, and descriptive exported identifiers. Keep command behavior in `internal/cli` and reusable content or rendering logic in its corresponding package. Name Go tests `TestXxx` in adjacent `_test.go` files. Keep browser assets offline and update `assets/vendor/manifest.json` and license notices when vendoring a dependency.

## Testing Guidelines

Add focused regression tests for link resolution, mounts, security checks, publishing selection, and output staleness when changing those paths. Browser rendering tests use headless Chromium when available; set `KB_BROWSER` explicitly in CI. CI runs on macOS and Linux and also builds the CLI and checks the sample workspace.

## Commit & Pull Request Guidelines

No fixed commit-message convention exists. Use a concise imperative subject. **Never commit or push directly to `main`.** Committing and pushing to a feature branch is fine without asking; changes reach `main` only through a reviewed pull request. In pull requests, explain behavior and risk, note relevant issues and tests, and include screenshots for visible viewer changes.

## Security & Configuration

`render.trust_html` is false by default; enable it only for fully trusted notes. Preserve workspace path and symlink containment, validate SVGs, and never serve active linked files in safe mode. Treat `sync.command` as trusted shell configuration, not an automatic build step.

## Agent Communication

No meta commentary: do not narrate internal reasoning, tool use, or routine progress. Give direct answers and concise change/test summaries; mention blockers only when relevant.
