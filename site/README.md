# kb

**Offline-first Markdown knowledge bases.** `kb` is a single command-line tool
that captures notes, checks links, searches your files, serves a browsable
wiki, and packs the whole thing into HTML files that open anywhere, without a
server, a CDN, or an account. Your Markdown stays the source of truth.

> This site is itself a `kb` workspace: the `site/` folder of the repository,
> built with `kb build`.

## Why kb

- **Plain files.** Notes are ordinary Markdown in ordinary folders. No
  database, no proprietary format, nothing to migrate away from.
- **Checked like code.** `kb check` fails on broken links, missing heading
  anchors, and pages without exactly one title, so a wiki stays navigable as
  it grows.
- **Works offline.** Search, viewers, Mermaid diagrams, and syntax
  highlighting ship inside the binary. Nothing is fetched at view time.
- **Portable output.** `kb publish` writes one self-contained `wiki.html`
  (plus one file per folder) that you can email, archive, or open from a USB
  stick.
- **Safe by default.** Rendered HTML is sanitized, SVGs are validated, and
  active files such as scripts are never served unless you opt in.

## Thirty-second tour

```sh
go install github.com/ankurs/kb/cmd/kb@latest
cd ~/notes
kb init                        # create kb.yaml
kb capture "Look into reminder timing"
kb check                       # validate headings and links
kb serve                       # browse at http://127.0.0.1:3000
kb publish                     # write dist/wiki.html
```

## Documentation

- [Install](start/install.md) — Go install or prebuilt release archives.
- [Quick start](start/README.md) — a first workspace in five minutes.
- [Core concepts](start/concepts.md) — workspaces, pages, mounts, and outputs.
- [Guides](guides/README.md) — capture, viewing, publishing, search, mounts.
- [Command reference](reference/commands.md) — every command and flag.
- [Configuration reference](reference/configuration.md) — every `kb.yaml` key.
- [Security model](reference/security.md) — what safe mode protects against.
- [Example workspace](example.md) — a complete fictional wiki to explore.
- [FAQ](faq.md) — common questions and comparisons.
- [Contributing](contributing.md) — build, test, and send changes.

## Project

`kb` is open source under the Apache-2.0 license. Source, issues, and
releases live at [github.com/ankurs/kb](https://github.com/ankurs/kb).
