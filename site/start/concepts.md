# Core concepts

## Workspace

A directory with a `kb.yaml`. Its content root (`root`, default `.`) holds
the Markdown tree. Generated output lives under `.kb/` and `dist/` by default
and is excluded from the content.

## Pages

Every `.md` file that is not excluded is a page. A page's title is its single
`# H1` heading. A folder's `README.md` is its overview page and supplies the
folder's label in navigation. Files such as images, CSV, or text that pages
link to are **linked resources**.

## Navigation

The sidebar mirrors the folder tree; there is no separate table of contents
to maintain. Folder `README.md` pages come first, root pages follow the order
of links in the root `README.md`, and `navigation.overrides` can retitle,
reorder, or hide any path.

## Mounts

A mount presents another Markdown tree, such as a shared handbook, under a
name inside the wiki. A mount can name an HTML fallback that supplies its
pages when the source directory is unavailable. See [Mounts](../guides/mounts.md).

## Outputs

`kb` produces three viewers from the same content:

| Viewer | Command | Needs a server | Typical use |
| --- | --- | --- | --- |
| Live viewer | `kb serve` | Yes, built in | Writing and reviewing locally |
| Static site | `kb build` | Any static host | Hosting a team or public wiki |
| Single-file bundles | `kb build`, `kb publish` | No | Sharing, archiving, offline use |

`kb build` writes the static site and snapshot bundles under `.kb/`.
`kb publish` writes a separate, HTML-only distribution under `dist/`. All
three share the same navigation, search, and safe rendering rules.

## Safe mode

By default, rendered Markdown is sanitized, Mermaid runs in strict mode, SVGs
are validated, and links to active files are rejected. `render.trust_html:
true` lifts these limits for fully trusted authors only. See the
[security model](../reference/security.md).

## Configuration layers

Settings come from built-in defaults, then `kb.yaml`, then user settings in
`~/.config/kb/config.yaml`, then command-line flags. See the
[configuration reference](../reference/configuration.md#precedence).
