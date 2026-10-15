# Viewing and building

## Live viewer

```sh
kb serve                         # http://127.0.0.1:3000
kb serve --port 3001
kb serve --host 0.0.0.0          # expose on your network; use with care
```

The live viewer re-reads the workspace as you edit and reloads the browser
when files change. It binds to `127.0.0.1` unless you change
`server.host` or pass `--host`.

## Static site and snapshots

```sh
kb build
```

`kb build` writes:

- **A static site** to `output.site` (default `.kb/site`): an `index.html`,
  your Markdown, linked resources, and the viewer assets. It must be served
  over HTTP, from the root of a host or domain.
- **Snapshots** to `output.bundles` (default `.kb/snapshots`): `wiki.html`
  with every page, plus one bundle per top-level folder. Snapshots open
  directly through `file://`.

Images referenced by pages are embedded in the snapshots. Links to other
local files resolve relative to the snapshot, so those files must be present
beside it. `kb build` only replaces a site directory that it created, which it
marks with a `.kb-generated` file.

## Verifying output

```sh
kb verify
```

`kb verify` runs the same checks as `kb check`, then confirms the snapshots on
disk match what `kb build` would produce now, without writing anything. Use it
in CI when snapshots are committed, to catch output that is out of date.

## How navigation is built

- The sidebar follows the folder tree. Sections are collapsed except for the
  ancestors of the current page.
- A folder's label is the H1 of its `README.md`, or a prettified folder name.
- Leading qualifiers that repeat a parent label are dropped, so a
  "Tool Library — Meetings" folder under "Tool Library" reads as "Meetings".
- Within a folder, `README.md` appears first as **Overview**; other pages sort
  by path.
- At the root, pages come before folders, in the order the root `README.md`
  links to them.
- A folder containing only one subfolder is merged into a single
  "Parent / Child" entry.

Override any of this per path:

```yaml
navigation:
  overrides:
    projects: {title: Active projects, order: 1}
    projects/archive: {hide: true}
    NOW.md: {title: Today}
```

Lower `order` values sort first. Hidden pages remain in the wiki and can still
be linked; they are only omitted from the sidebar.

## Viewer features

All three viewers share a top bar, collapsible responsive navigation,
active-page highlighting, ranked search over titles and content (Cmd/Ctrl+K,
Escape to return), Mermaid diagrams, and syntax highlighting for common
languages. Web links open in a new tab unless
`render.external_links_new_tab: false`.
