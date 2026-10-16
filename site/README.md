# kb

**Offline-first Markdown knowledge bases.** `kb` is a single command-line tool
that captures notes, checks links, searches your files, serves a browsable
wiki, and packs the whole thing into HTML files that open anywhere, without a
server, a CDN, or an account. Your Markdown stays the source of truth; see
[Why a folder of Markdown](start/why-markdown.md) for the case for plain files.

> This site is itself a `kb` workspace: the `site/` folder of the repository,
> built with `kb build`.

## Why kb

- **Checked like code.** `kb check` fails on broken links, missing heading
  anchors, and pages without exactly one title, so a wiki stays navigable as
  it grows.
- **Works offline.** Search, viewers, Mermaid diagrams, Excalidraw PNG
  previews, and syntax highlighting ship inside the binary and generated
  HTML. Only content your notes link to by URL, such as remote images, is
  fetched at view time.
- **Portable output.** `kb publish` writes one self-contained `wiki.html`
  (plus one file per folder) that you can email, archive, or open from a USB
  stick.
- **Ready for coding agents.** Claude Code, Codex, Mistral Vibe, and pi can
  read and edit your notes, run `kb check` as a guardrail, and search them
  locally through qmd's MCP server. See
  [Use with coding agents](guides/coding-agents.md).
- **Safe by default.** Rendered HTML is sanitized, SVGs are validated, and
  active files such as scripts are never served unless you opt in.

## Thirty-second tour

```sh
go install github.com/ankurs/kb/cmd/kb@latest  # needs Go 1.24+
cd ~/notes
kb init                        # create kb.yaml
kb capture "Look into reminder timing"
kb check                       # validate headings and links
kb serve                       # browse at http://127.0.0.1:3000
kb publish                     # write dist/wiki.html
```

## Documentation

**Get started**

- [Why a folder of Markdown](start/why-markdown.md) — the case for plain files.
- [Install](start/install.md) — `go install`, release archives, or source.
- [Quick start](start/README.md) — a first workspace in five minutes.
- [Core concepts](start/concepts.md) — workspaces, pages, mounts, and outputs.
- [Example workspace](start/example.md) — a complete fictional wiki to explore.

**Guides**

- [Capture and review](guides/capture.md) — the inbox, templates, and review loop.
- [Viewing and building](guides/viewing.md) — live viewer, static site, snapshots.
- [Publishing and sync](guides/publishing.md) — single-file HTML distributions.
- [Search](guides/search.md) — keyword, browser, and qmd hybrid search.
- [Use with coding agents](guides/coding-agents.md) — agents, checks, and MCP search.
- [Mounts](guides/mounts.md) — sharing a Markdown tree across workspaces.
- [Automating checks](guides/automation.md) — hooks, CI, and hosting.

**Reference**

- [Commands](reference/commands.md) — every command and flag.
- [Configuration](reference/configuration.md) — every `kb.yaml` key.
- [Security model](reference/security.md) — what safe mode protects against.

**Project**

- [FAQ](faq.md) — common questions and comparisons.
- [Contributing](contributing.md) — build, test, and send changes.

## Project

`kb` is open source under the Apache-2.0 license. Source, issues, and
releases live at [github.com/ankurs/kb](https://github.com/ankurs/kb).
