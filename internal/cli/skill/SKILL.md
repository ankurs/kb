---
name: kb
description: "Work in a kb Markdown knowledge-base workspace to find, capture, create, and validate notes. Use when asked to maintain or browse a kb workspace, not for unrelated Markdown projects."
---

# Work in a kb workspace

Find the workspace's `kb.yaml` before running content commands. `kb` discovers it from the current directory or its parents; use `kb --workspace PATH` when working elsewhere. Markdown files are the source of truth; `.kb/` and `dist/` are generated output.

- Search for existing material before creating or rewriting notes. `kb find "terms"` searches authored Markdown without an index. If qmd is configured and indexed, `kb search "question"` gives ranked results and snippets; read the source Markdown before relying on a hit or editing it. `kb doctor` helps diagnose optional qmd setup. Do not run `kb index` merely to read notes: it may invoke models and download data.
- For a loose item, use `kb capture --at <project> "text"`; use `kb review` to inspect the inbox. For a new note, use `kb new page --at <project> --title "..."` or `kb new meeting --at <project> --title "..."` and edit the created Markdown. Respect the workspace's existing organization and instructions.
- After changing notes, run `kb check` and address broken local links, anchors, and headings. If the task includes maintaining already-generated output, use `kb build` and then `kb verify`; `kb verify` reports stale output but does not write it.
- Run `kb publish` only when the user wants a distribution. Never run `kb sync` automatically: it executes a trusted shell command from configuration and may affect destinations outside the workspace. `kb sync --show` displays that command without executing it. Do not enable `render.trust_html` just to display untrusted content.

`kb` and qmd are complementary: qmd can retrieve indexed documents through its CLI or MCP server; use `kb` for workspace-specific content workflows and validation. If qmd is unavailable, continue with `kb find` and the Markdown files.
