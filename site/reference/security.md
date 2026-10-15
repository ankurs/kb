# Security model

`kb` is designed so that opening someone else's notes is safe by default.

## Safe mode (default)

With `render.trust_html: false`, in every viewer:

- **HTML is sanitized.** Ordinary inline HTML such as `<br>` and tables is
  kept; scripts, event handlers, and unsafe URLs are removed by the bundled
  DOMPurify. The Markdown renderer does not sanitize on its own.
- **Mermaid runs in strict mode**, and diagram source is escaped before
  rendering.
- **SVGs are validated.** Linked SVG files must use safe drawing elements and
  attributes, or the build fails. This applies in trusted mode too.
- **Active files are refused.** Links to HTML, scripts, stylesheets, and
  similar files are errors in `kb check`, are not served by `kb serve`, and
  are not copied into the site.

## Trusted mode

`render.trust_html: true` restores raw HTML and permits links to active files.
Enable it only when **every** author of the notes is trusted. It follows the
normal configuration precedence, so a user setting can enable it for one
workspace.

## Containment

Paths in configuration and links must stay inside the content root; the only
exception is `publish.dir`, which may be an absolute path outside it. `kb`
refuses to read through symlinks that leave the content root, to write
through symlinked outputs, and to replace directories it did not create.

## Network

- Viewer code and assets (search, Markdown rendering, Mermaid, highlighting)
  are bundled; no viewer loads them from a CDN.
- Content your notes reference by URL, such as a remote image, is not
  bundled. Viewing the page requests it from its host, and it is unavailable
  offline. Keep images local if a wiki must work offline or must not contact
  other hosts.
- `kb serve` binds to `127.0.0.1` unless you choose otherwise.
- qmd may download models the first time it indexes or searches, and your own
  `sync.command` may use the network. No other `kb` command does.

## Sync is trusted configuration

`sync.command` is an arbitrary shell command. `kb` never runs it implicitly:
not during `build`, `publish`, or anything else. Inspect it with
`kb sync --show` before running a workspace you did not write.

## Reporting a vulnerability

Please report security issues privately through
[GitHub security advisories](https://github.com/ankurs/kb/security/advisories/new)
rather than in a public issue.
