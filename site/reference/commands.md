# Commands

## Global flags

| Flag | Meaning |
| --- | --- |
| `-w`, `--workspace PATH` | Workspace to use. Default: search the current directory and its parents for `kb.yaml`. |
| `--global-config PATH` | User settings file. Default: `~/.config/kb/config.yaml`. |
| `-v`, `--verbose` | Print progress diagnostics to stderr. Standard output is unchanged. |

## Content

### kb init

Create a minimal `kb.yaml` in the workspace directory. Never overwrites an
existing file.

### kb capture TEXT

Append a dated item to the inbox's `## Unprocessed` section.

| Flag | Meaning |
| --- | --- |
| `--at PATH` | Project label written in brackets before the text. |

### kb new meeting|page

Create a page from a template; refuses to overwrite.

| Flag | Meaning |
| --- | --- |
| `--at PATH` | Required. Project directory relative to the content root. |
| `--title TEXT` | Required. Page title; also used for the filename. |
| `--date YYYY-MM-DD` | Meeting date. Default: today. |

### kb review

List unprocessed inbox items without changing the file.

### kb check

Validate headings, links, anchors, and linked files. Prints
`page:line: level: message` per issue and a summary line. Exits non-zero
when any error is found; warnings do not fail.

## Search

### kb find QUERY

Keyword search over Markdown without an index.

| Flag | Meaning |
| --- | --- |
| `--scope PREFIX` | Limit to pages under a path prefix. |

### kb search QUERY

Hybrid retrieval through qmd. Requires `search.provider: qmd`.

| Flag | Meaning |
| --- | --- |
| `--scope NAME` | qmd collection to search. |

### kb index

Run `qmd update` and `qmd embed`.

| Flag | Meaning |
| --- | --- |
| `--force` | Re-embed every document. |

## Output

### kb serve

Serve the live viewer with automatic navigation and reload.

| Flag | Meaning |
| --- | --- |
| `--host ADDR` | Bind address. Default: `server.host` (`127.0.0.1`). |
| `-p`, `--port N` | Port. Default: `server.port` (`3000`). |

### kb build

Write the static site and snapshot bundles. Prints each output path and any
warnings.

### kb verify

Run `kb check`'s validations, then confirm the snapshots match what
`kb build` would write. Writes nothing; exits non-zero on the first error.

### kb publish

Write an HTML-only distribution.

| Flag | Meaning |
| --- | --- |
| `-o`, `--output DIR` | Distribution directory (relative to the workspace or absolute). |
| `--wiki NAME` | Filename for the combined wiki. |
| `--include LIST` | Top-level folders to include; repeat or comma-separate. |
| `--exclude LIST` | Top-level folders to omit; repeat or comma-separate. |
| `--root-pages` | Include root pages when folder filters are set. |

### kb sync

Run `sync.command` from the workspace root.

| Flag | Meaning |
| --- | --- |
| `--show` | Print the working directory, publish directory, and command without running it. |

## Diagnostics

### kb doctor

Print the workspace root and config path, then any optional qmd
configuration that is missing or needs trust.
