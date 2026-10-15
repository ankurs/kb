# Configuration

A workspace's `kb.yaml` needs only `version: 1`; everything else has a
default. Unknown keys are rejected.

## Full example

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
navigation:
  overrides:
    tool-library: {title: Tool Library Pilot, order: 1}
    tool-library/drafts: {hide: true}
check:
  exclude: ['**/docs/**']
  warn_missing: ['handbook/**']
render:
  trust_html: false
  external_links_new_tab: true
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

## Keys

| Key | Default | Meaning |
| --- | --- | --- |
| `version` | `1` | Configuration format version. |
| `id` | — | Stable identifier; selects `workspaces.<id>` in user settings. |
| `name` | — | Title shown in viewers and bundles. |
| `root` | `.` | Content root, relative to `kb.yaml`. |
| `exclude` | see below | Glob patterns removed from the content. Replaces the default list when set. |
| `mounts[].name` | — | Path at which the mounted tree appears. |
| `mounts[].path` | — | Directory holding the mounted Markdown. |
| `mounts[].fallback` | — | HTML bundle used when `path` is unavailable. |
| `navigation.overrides.<path>` | — | `title`, `order`, and/or `hide` for a folder or page. |
| `check.exclude` | `[]` | Pages skipped by `kb check` but still navigable and bundled. |
| `check.warn_missing` | `[]` | Pages whose missing links are warnings, not errors. |
| `render.trust_html` | `false` | Allow raw HTML and active linked files. Trusted authors only. |
| `render.external_links_new_tab` | `true` | Open web links in a new tab with `rel="noopener noreferrer"`. |
| `output.site` | `.kb/site` | Static site directory written by `kb build`. |
| `output.bundles` | `.kb/snapshots` | Snapshot directory written by `kb build`. |
| `publish.dir` | `dist` | Distribution directory for `kb publish`. |
| `publish.wiki` | `wiki.html` | Filename of the combined wiki. |
| `publish.include` | — | Top-level folders to publish. |
| `publish.exclude` | — | Top-level folders to omit. |
| `publish.root` | `false` | Keep root pages when `include` or `exclude` is set. |
| `publish.names.<folder>` | — | Filename for a folder's focused bundle. |
| `sync.command` | — | Shell command run by `kb sync`. |
| `capture.inbox` | `inbox.md` | Inbox file for `kb capture` and `kb review`. |
| `capture.templates` | `.templates` | Template folder for `kb new`. |
| `search.provider` | `qmd` | Provider for `kb search` and `kb index`. |
| `search.qmd_config` | `.qmd` | qmd configuration directory. |
| `server.host` | `127.0.0.1` | Bind address for `kb serve`. |
| `server.port` | `3000` | Port for `kb serve`. |

Default `exclude` list: `.git/**`, `.kb/**`, `.qmd/**`, `.agents/**`,
`.claude/**`, `.obsidian/**`, `.plans/**`, `**/.plans/**`, `scripts/**`,
`vendor/**`, `dist/**`, `derived/**`, `**/AGENTS.md`, `**/CLAUDE.md`,
`**/_sidebar.md`, and `**/_coverpage.md`.

## User settings

`~/.config/kb/config.yaml` can set defaults for every workspace and overrides
for one workspace by its `id`:

```yaml
defaults:
  server: {port: 4000}
workspaces:
  personal-notes:
    output: {site: .kb/preview}
```

## Precedence

From lowest to highest: built-in defaults, `kb.yaml`, user `defaults`, user
`workspaces.<id>`, then command-line flags. Maps merge key by key; lists
replace the earlier value entirely.
