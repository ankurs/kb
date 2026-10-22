# kb

`kb` is an offline-first CLI for Markdown knowledge bases. It captures notes,
checks links, searches your files, serves a browsable wiki, and packs the
whole thing into HTML files that open anywhere, without a server, a CDN, or an
account. Markdown files remain the source of truth.

**Documentation:** [ankurs.github.io/kb](https://ankurs.github.io/kb/)

- **Checked like code.** `kb check` fails on broken links, missing heading
  anchors, and pages without exactly one title.
- **Works offline.** Search, viewers, Mermaid diagrams, Excalidraw PNG
  previews, and syntax highlighting ship inside the binary and generated HTML.
- **Portable output.** `kb publish` writes one self-contained `wiki.html`, plus
  one file per folder, that you can email, archive, or open from a USB stick.
- **Ready for coding agents.** Agents can edit notes, run `kb check` as a
  guardrail, and search locally through qmd's MCP server.
- **Safe by default.** Rendered HTML is sanitized, SVGs are validated, and
  active files such as scripts are never served unless you opt in.

## Install

Requires Go 1.24 or later:

```sh
go install github.com/ankurs/kb/cmd/kb@latest
```

The binary goes to `$(go env GOPATH)/bin`; make sure that directory is on your
`PATH`. Prebuilt macOS and Linux archives are on the
[releases page](https://github.com/ankurs/kb/releases). See
[Install](https://ankurs.github.io/kb/#/start/install.md) for checksums,
pinning a version, and building from source.

## Quick start

From a folder of Markdown:

```sh
kb init                         # create kb.yaml if needed
kb check                        # validate headings and local links
kb serve                        # browse at http://127.0.0.1:3000
kb build                        # create the static site and offline bundles
kb verify                       # check that generated output is current
kb publish                      # write dist/wiki.html and focused bundles
```

`kb` searches parent directories for `kb.yaml`; use `--workspace /path/to/notes`
to select a workspace explicitly. `kb init` never overwrites an existing file.

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
| `kb serve --port 3000` | Serve the wiki with live reload and optional qmd-backed search. |
| `kb build` | Build a static site and local HTML snapshots. |
| `kb publish` | Build an HTML-only distribution with `wiki.html` and a bundle per folder. |
| `kb sync` | Run the trusted sync command from configuration. |
| `kb verify` | Check content and generated output without writing files. |
| `kb doctor` | Show workspace settings and optional qmd diagnostics. |
| `kb skill show` / `kb skill install` | Display or install the bundled agent skill. |

Add `-v` or `--verbose` to any command for progress diagnostics on stderr.
Every flag is in the [command reference](https://ankurs.github.io/kb/#/reference/commands.md).

## Agent skill

`kb skill show` prints the bundled [kb skill](internal/cli/skill/SKILL.md) without
requiring a workspace. From the directory where you want project-scoped agent
instructions, run `kb skill install` to write `.agents/skills/kb/SKILL.md`.
Use `kb skill install --global` for `~/.agents/skills/kb/SKILL.md`. Installation
refuses to replace an existing skill file unless you pass `--force` (`-f`); it
does not edit `AGENTS.md`, configure qmd, or run indexing. See
[coding agents](https://ankurs.github.io/kb/#/guides/coding-agents.md) for the
workflows it covers.

## Try the example

[`testdata/sample`](testdata/sample/README.md) is a fictional tool-lending
pilot with journal entries, meetings, weekly reviews, a decision log, Mermaid
and SVG diagrams, an Excalidraw scene, a mounted handbook, and qmd
collections. To look without installing anything, download the
[prebuilt example](examples/wiki.html) and open it in a browser.

To work with it, copy it somewhere disposable so the fixture stays unchanged:

```sh
git clone https://github.com/ankurs/kb.git && cd kb
demo=$(mktemp -d)
rsync -a --exclude '.kb/' --exclude 'dist/' --exclude '.qmd/index.sqlite*' testdata/sample/ "$demo/"
cd "$demo"
kb review
kb capture --at tool-library "Confirm where returned tools wait for inspection"
kb check && kb build && kb verify && kb publish
kb sync --show                  # inspect the local-only sync command
kb serve
```

## Documentation

- [Why a folder of Markdown](https://ankurs.github.io/kb/#/start/why-markdown.md)
  and [core concepts](https://ankurs.github.io/kb/#/start/concepts.md)
- Guides: [capture](https://ankurs.github.io/kb/#/guides/capture.md),
  [viewing and building](https://ankurs.github.io/kb/#/guides/viewing.md),
  [publishing and sync](https://ankurs.github.io/kb/#/guides/publishing.md),
  [search](https://ankurs.github.io/kb/#/guides/search.md),
  [coding agents](https://ankurs.github.io/kb/#/guides/coding-agents.md),
  [mounts](https://ankurs.github.io/kb/#/guides/mounts.md),
  [automation](https://ankurs.github.io/kb/#/guides/automation.md)
- Reference: [configuration](https://ankurs.github.io/kb/#/reference/configuration.md)
  and [security model](https://ankurs.github.io/kb/#/reference/security.md)
- [FAQ](https://ankurs.github.io/kb/#/faq.md)

The site source is the [`site/`](site/README.md) folder, itself a `kb`
workspace, so the same pages are readable offline in this repository.

## Development

```sh
go test ./...
go vet ./...
go run ./cmd/kb --workspace testdata/sample check
go run ./cmd/kb --workspace site serve     # preview the documentation
```

CI runs formatting, `go vet`, race-enabled tests, and sample-workspace checks
on macOS and Linux. Tags matching `v*` publish release archives with SHA-256
checksums, and the Pages workflow deploys `site/`. See
[Contributing](site/contributing.md).

The source code is licensed under Apache-2.0. Browser asset licenses are in
`assets/vendor/licenses/`.
