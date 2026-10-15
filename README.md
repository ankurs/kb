# kb

`kb` is an offline-first CLI for Markdown knowledge bases. It provides capture,
content checks, local search, a browser viewer, and portable HTML bundles.
Markdown files remain the source of truth.

## Install

Requires Go 1.24 or later.

```sh
go install ./cmd/kb
```

From the source directory, `go run ./cmd/kb` also works without installation.

## Quick start

Run these commands from a Markdown repository:

```sh
kb init                         # create kb.yaml if needed
kb check                        # validate headings and local links
kb serve                        # browse at http://127.0.0.1:3000
kb build                        # create the site and offline HTML bundles
kb verify                       # check that generated output is current
kb publish                      # build dist/wiki.html and focused bundles
```

`kb` searches parent directories for `kb.yaml`. Use `--workspace /path/to/notes`
to select a workspace explicitly. `kb init` does not overwrite an existing file.

## Commands

| Command | Purpose |
| --- | --- |
| `kb capture --at tool-library "Follow up"` | Add a dated item to `inbox.md`. |
| `kb new meeting --at tool-library --title "Design review" --date 2026-10-10` | Create a meeting note from a template. |
| `kb new page --at tool-library --title "Architecture"` | Create a knowledge-base page from a template. |
| `kb review` | List unprocessed inbox items. |
| `kb check` | Validate Markdown headings and local links. |
| `kb find --scope tool-library "seven calendar days"` | Search Markdown by keyword. |
| `kb search --scope wiki "borrowing policy"` | Hybrid search over a local qmd collection. |
| `kb index` | Run `qmd update` and `qmd embed`. |
| `kb serve --port 3000` | Serve the wiki with automatic navigation and browser reload. |
| `kb build` | Build a static site and local HTML snapshots. |
| `kb publish` | Build an HTML-only distribution with `wiki.html` and a bundle per folder. |
| `kb sync` | Run the sync command from configuration. |
| `kb verify` | Check content and generated output without writing files. |
| `kb doctor` | Show workspace settings and optional qmd diagnostics. |

Use `-v` or `--verbose` with any command to print progress diagnostics to
stderr, for example `kb -v build` or `kb check --verbose`. Standard output
keeps the same format.

## Try a complete workflow

[`testdata/sample`](testdata/sample/README.md) is a fictional tool-lending
pilot used every day from **11 September to 10 October 2026**. Thirty dated
[journal entries](testdata/sample/tool-library/journal/README.md), nine
[meetings](testdata/sample/tool-library/meetings/README.md), four weekly reviews,
a source register, and a current policy show the capture → evidence → decision
→ status loop. It also demonstrates Mermaid and SVG diagrams, a mounted
handbook, qmd collections, offline builds, publishing, and a safe local sync.

From this source directory, with `kb` installed:

```sh
demo=$(mktemp -d)
rsync -a --exclude '.kb/' --exclude 'dist/' --exclude '.qmd/index.sqlite*' testdata/sample/ "$demo/"
cd "$demo"
kb review
kb find --scope tool-library "seven calendar days"
kb capture --at tool-library "Confirm where returned tools wait for inspection"
kb new meeting --at tool-library --title "Friday Desk Review" --date 2026-10-10
kb check
kb build
kb verify
kb publish
kb sync --show                    # inspect before running the local copy
kb sync                           # copy dist/ to .kb/mirror/
kb serve
```

The build writes a static site to `.kb/site` and a complete offline snapshot
to `.kb/snapshots/wiki.html`. Open the snapshot directly or browse the served
site at `http://127.0.0.1:3000`.

`kb serve` binds to `127.0.0.1` by default. Navigation comes from the Markdown
tree; Docsify receives a virtual `_sidebar.md`, including on nested routes.
Top-level sections are expanded by default. Section labels come from their
`README.md` headings, repeated ancestor labels are shortened, and root pages
follow links in the root README. `navigation.overrides` can set a title, order,
or hide a path. The viewer uses locally bundled browser assets and does not
require a CDN.

`kb build` writes a static site to `.kb/site` and `wiki.html` plus a bundle for
each top-level folder to `.kb/snapshots`. The static site needs an HTTP server;
the bundles open directly through `file://`. Output paths are configurable.
Images referenced by Markdown are embedded in the bundles. Links to other local
files resolve relative to the bundle and require those files to be present.
In the default safe mode, links to active files such as HTML, scripts, and
stylesheets are errors rather than files served or copied into the site.

`kb publish` builds a separate distribution without running `kb build` or
`kb sync`. By default it writes `dist/wiki.html` and one focused HTML file per
top-level folder. Markdown and referenced images are embedded; non-image source
files are not copied, so their links are inactive in the published HTML. Only
a directory previously created by `kb publish` (marked by `.kb-published`)
is replaced. Move an existing, unrelated `dist/` aside before publishing
there. Set `publish.dir` to a relative path or an absolute destination outside
the workspace; use
`kb publish --output /path/to/dist` to override it for one run.

`kb sync` runs `sync.command` from the workspace root using `sh -c`. The command
receives `KB_WORKSPACE_ROOT` and `KB_PUBLISH_DIR` as absolute environment
variables. It does not publish or rebuild first. `kb sync --show` prints the
configured command without executing it; it is not a simulation of the
command's effects. Run `kb sync` only with a trusted configuration.

## Configuration

A `kb.yaml` containing `version: 1` uses the defaults. For example:

```yaml
version: 1
id: personal-notes
name: Personal Notes
root: .
mounts:
  - name: handbook
    path: reference
    fallback: .kb/snapshots/handbook.html
exclude:
  - .git/**
  - .kb/**
  - derived/**
  - scripts/**
  - '**/.plans/**'
navigation:
  overrides:
    tool-library: {title: Tool Library Pilot, order: 1}
    tool-library/drafts: {hide: true}
check:
  exclude: ['**/docs/**']
  warn_missing: ['handbook/**']
render:
  trust_html: false  # set true only for fully trusted authors
output:
  site: .kb/site
  bundles: .kb/snapshots
publish:
  dir: dist
  wiki: wiki.html
  names:
    tool-library: borrowing-pilot.html
sync:
  command: >-
    rsync -av -- "$KB_PUBLISH_DIR/" "$HOME/wiki-backup/"
capture:
  inbox: inbox.md
  templates: .templates
search:
  provider: qmd
  qmd_config: .qmd
server:
  host: 127.0.0.1
  port: 3000
```

Rendered Markdown is sanitized by default in the served, static, and single-file
viewers. This keeps ordinary inline HTML (such as `<br>` and tables), but
removes scripts, event handlers, and unsafe URLs. Mermaid uses strict security
mode. SVG files linked by pages must use safe drawing elements and attributes.
Set `render.trust_html: true` only if every note author is trusted: it restores
raw HTML in both viewers and permits linked active files; it must **not** be
used for untrusted notes. SVG assets are still validated in trusted mode. This
setting follows the normal repository → user-global precedence. Marked itself
does not sanitize HTML; the bundled DOMPurify runtime does.

`exclude` replaces the default exclusion list when specified. `check.exclude`
omits pages from validation while leaving them available for navigation and
bundling; `check.warn_missing` reports missing links as warnings for matching
pages. Navigation overrides set a path's title, order, or visibility.

`publish.include` selects top-level folders; `publish.exclude` removes them.
Both affect the combined wiki and focused bundles. With neither set, all
folders and root pages are included. When either is set, root pages are omitted
unless `publish.root: true`. `publish.names` renames focused HTML files, and
`publish.wiki` names the combined file. The equivalent command-line overrides
are `--include`, `--exclude`, `--root-pages`, `--wiki`, and `--output`.
Publishing validates only selected pages and their linked resources; an
excluded unavailable mount does not block publication of other folders.

Mounts expose another Markdown tree under a named path. Absolute Markdown links
within a mount resolve from that mount's root. If a mount is unavailable, its
configured HTML fallback supplies pages to the combined bundle. A missing
mount and fallback cause the build to fail.

User settings live in `~/.config/kb/config.yaml`. They may set `defaults` and
`workspaces.<id>` values:

```yaml
defaults:
  server: {port: 4000}
workspaces:
  personal-notes:
    output: {site: .kb/preview}
```

Precedence is built-in defaults, `kb.yaml`, user defaults, workspace-specific
user settings, then CLI flags. Maps merge by key; lists replace. Use
`--global-config` to select a different user settings file.

`kb find` searches Markdown by keyword without an index or model. `kb search`
uses qmd hybrid retrieval and returns ranked documents and snippets, not a
synthesized answer. `kb index` updates qmd and generates embeddings.
The sample includes [`.qmd/index.yml`](testdata/sample/.qmd/index.yml) with
`wiki` and `handbook` collections, but not its SQLite database or models. To
set up local semantic retrieval on the copied demo:

```sh
qmd init                         # creates the local SQLite index
kb index                         # update documents and embed (models download once)
kb search --scope wiki "when is the day-five reminder sent?"
kb search --scope handbook "how do I check in a returned tool?"
qmd bench .qmd/bench-tool-library.json -c wiki  # optional retrieval check
```

The config uses only in-workspace paths and default models. `kb find` and all
reading and publishing commands work without qmd, models, or network access.

`kb doctor` reports missing project configuration and empty collections. A
project config with update hooks, out-of-project collection paths, or custom
models requires approval with `qmd trust`; `kb` does not change trust settings.

## GitHub Actions

CI runs on pushes, pull requests, and manual dispatch on macOS and Linux. It
checks formatting and dependencies, runs `go vet` and race-enabled tests, builds
the CLI, and checks, builds, and verifies the sample workspace. Version tags
matching `v*` create macOS and Linux release archives with SHA-256 checksums.

## Development

```sh
go test ./...
go vet ./...
go run ./cmd/kb --workspace testdata/sample check
```

The source code is licensed under Apache-2.0. Browser asset licenses are in
`assets/vendor/licenses/`.
