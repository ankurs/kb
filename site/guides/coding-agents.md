# Use with coding agents

A `kb` workspace is a folder of Markdown files, which is the format coding
agents such as Claude Code, Codex, Mistral Vibe, and pi read and write most
naturally. There is no export step and no API: the agent opens the same files
you do, and `kb` keeps the result checked and browsable.

## Why the combination works

- **Agents already speak files.** They can read, search, and edit notes with
  their normal tools, and propose changes you review as a diff.
- **`kb check` is a guardrail.** Ask the agent to run it after every edit. It
  catches broken links, missing anchors, and pages without a single title
  before you see them.
- **Agent instructions stay out of the wiki.** `AGENTS.md`, `CLAUDE.md`,
  `.agents/`, and `.claude/` are excluded from navigation, search, and
  output by default, so you can keep agent guidance next to your notes.
- **Safe mode still applies.** Whatever an agent writes into a page, the
  viewers sanitize its HTML and refuse active files unless you have enabled
  `render.trust_html`.
- **Retrieval stays local.** With qmd, the agent can search your notes by
  meaning through an MCP server running on your machine.

## Tell the agent how to work

Put a short `AGENTS.md` (or `CLAUDE.md`) at the workspace root. For example:

```markdown
# Notes workspace

- This is a kb workspace. Pages are Markdown with exactly one H1.
- Find material with `kb find "terms"` or `kb search "question"` before writing.
- Record loose ideas with `kb capture --at <project> "text"`.
- Create pages with `kb new page|meeting --at <project> --title "..."`.
- Run `kb check` after every change and fix all errors.
- Never edit `.kb/`, `dist/`, or generated HTML.
```

Codex, Claude Code, Mistral Vibe, and pi all load project instruction files
like this automatically or on request.

## Give the agent semantic search with qmd

[qmd](https://github.com/tobi/qmd) is the local search engine behind
`kb search` and `kb index`. Besides its command line, it runs as an MCP
server with `qmd mcp`, exposing tools to query the index (`query`), fetch
documents (`get`, `multi_get`), and report index health (`status`). The agent
then retrieves relevant notes by meaning, not just by keyword, without
anything leaving your machine.

Set up the index once in the workspace (see [Search](search.md)):

```sh
qmd init
qmd collection add . --name wiki
kb index
```

`qmd collection add` registers the workspace as the `wiki` collection in
`.qmd/index.yml`; `kb index` refuses to run until at least one collection
exists. Add `ignore` entries for generated output such as `.kb/**` and
`dist/**` there, since the static site contains copies of your pages. The
example workspace's
[`.qmd/index.yml`](https://github.com/ankurs/kb/blob/main/testdata/sample/.qmd/index.yml)
shows a complete configuration.

Then register `qmd mcp` as a stdio server in your agent, run from the
workspace root.

**Claude Code**: add a project-scoped `.mcp.json`:

```json
{
  "mcpServers": {
    "qmd": { "command": "qmd", "args": ["mcp"] }
  }
}
```

**Codex**: add to `.codex/config.toml` or `~/.codex/config.toml`:

```toml
[mcp_servers.qmd]
command = "qmd"
args = ["mcp"]
```

**Mistral Vibe**: add to `.vibe/config.toml` or `~/.vibe/config.toml`:

```toml
[[mcp_servers]]
name = "qmd"
transport = "stdio"
command = "qmd"
args = ["mcp"]
```

**pi** and other agents: register the same `qmd mcp` command in the agent's
MCP configuration, or skip MCP and let the agent run `kb search` from its
shell. Any agent with shell access can use every `kb` command.

If your agent starts MCP servers outside the workspace, or the workspace
sets a custom `search.qmd_config`, set `QMD_CONFIG_DIR` to the absolute path
of that directory in the server's environment. This is the same variable
`kb` passes to qmd.

## Without qmd

Everything except semantic search works with no extra setup: the agent can
use `kb find` for keyword search, `kb review` to read the inbox, and
`kb check` to validate its work.
