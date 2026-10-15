# Why a folder of Markdown

`kb` assumes your knowledge base is a folder of Markdown files rather than an
app or a hosted service. This page explains why that is a good place to keep
notes, what a plain folder lacks, and when it is the wrong choice.

## What plain files give you

- **You own them.** Text files on your disk need no account, subscription, or
  export button. They will still open in decades, long after any particular
  app has changed or shut down.
- **Any editor works.** VS Code, Vim, Obsidian, iA Writer, or a phone app.
  Switching editors does not mean migrating your notes.
- **Real history.** Put the folder in git and every change has a diff, an
  author, and a date. You can branch, revert, and review notes in a pull
  request, the same way you review code.
- **Notes live beside the work.** A `docs/` folder in a repository changes in
  the same commit as the code it describes, so the two stay in step.
- **Ordinary tools apply.** `grep`, scripts, backups, and sync services such
  as git, Syncthing, or Dropbox already understand files. Nothing needs an
  API or a plugin.
- **Private by default.** Nothing leaves your machine unless you send it.
- **Coding agents work with them directly.** Claude Code, Codex, Mistral
  Vibe, pi, and similar tools read and edit Markdown natively, so your notes
  become working context with no export or integration. See
  [Use with coding agents](../guides/coding-agents.md).

## What a plain folder lacks

A folder of Markdown files becomes hard to use as it grows. `kb` adds the
missing pieces without changing the files:

| On its own, a folder has… | `kb` adds |
| --- | --- |
| No navigation beyond a file tree | A generated sidebar and a browsable wiki: `kb serve` |
| Links that silently break when pages move | Link, anchor, and title checks: `kb check` |
| Search limited to filenames or `grep` | Ranked search in every viewer, `kb find`, and optional semantic search |
| No easy way to share with someone who will not clone a repository | Single-file HTML that opens anywhere: `kb publish` |
| No habit for capturing loose ideas | A dated inbox and templates: `kb capture`, `kb new` |

## When it is not the right fit

- **Live co-editing.** If several people need to type in the same document at
  the same time, a collaborative editor will serve them better.
- **Teams that will not touch files.** Some readers and writers need a web
  editor. `kb` can publish to them, but authoring still happens in files.
- **Per-page permissions.** Access is whatever the folder or repository
  allows; there is no access control per page.
- **Structured records.** Inventories, CRM data, and anything you sort and
  filter belong in a database or spreadsheet. Link to them from your notes.

If plain files suit you, continue to [Install](install.md).
