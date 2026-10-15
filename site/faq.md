# FAQ

## Does kb change my Markdown?

Only when you ask it to. `kb capture` appends to the inbox and `kb new`
creates new files; every other command only reads your content. Generated
output goes to `.kb/` and `dist/`.

## Does it need a network connection?

No. Every viewer, including search, diagrams, and highlighting, is embedded
in the binary and the generated HTML. Remote content your notes link to by
URL, such as an image hosted elsewhere, is still loaded from its host and is
unavailable offline; keep images in the workspace to avoid that. Otherwise,
only optional qmd model downloads and your own `sync.command` touch the
network.

## How is this different from a static site generator?

Static site generators build a website from Markdown. `kb` also does that, but
it is built around the notes themselves: capture and templates for writing,
link and anchor checks for keeping a wiki healthy, keyword and semantic
search, and single-file HTML that works with no server at all.

## Can I use it with Obsidian or another editor?

Yes. `kb` reads ordinary Markdown with ordinary inline links, and excludes
`.obsidian/` by default. Wiki-style `[[links]]` are not resolved.

## Does it work with AI coding agents?

Yes. Agents such as Claude Code, Codex, Mistral Vibe, and pi read and edit
the Markdown directly, can run `kb check` to validate their changes, and can
search your notes locally through qmd's MCP server. See
[Use with coding agents](guides/coding-agents.md).

## Can I host the wiki publicly?

Yes. Upload the `kb build` site to any static host, served from the root of a
domain or subdomain. See [Automating checks](guides/automation.md#hosting-the-static-site).
This documentation is hosted that way.

## Is it safe to open a stranger's wiki?

In the default safe mode, rendered HTML is sanitized and active files are
refused. Do not enable `render.trust_html`, and read `sync.command` before
running `kb sync`. See the [security model](reference/security.md).

## Which platforms are supported?

Release archives are built for macOS and Linux on `amd64` and `arm64`. Other
platforms supported by Go can build from source.
