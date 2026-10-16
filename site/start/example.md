# Example workspace

The repository includes a complete fictional wiki in
[`testdata/sample`](https://github.com/ankurs/kb/tree/main/testdata/sample):
a neighborhood tool-lending pilot that ran every day from 11 September to
10 October 2026. It is both documentation and a test fixture.

## What it shows

- Thirty dated journal entries, nine meetings, and four weekly reviews.
- A source register, a decision log, an action register, and a current
  policy that links back to the evidence behind it.
- Mermaid diagrams, a validated SVG, an editable Excalidraw scene with an
  offline PNG preview, and highlighted YAML and JSON.
- A mounted volunteer handbook with an HTML fallback.
- qmd collections and retrieval benchmarks.
- A safe, local-only `sync.command`.

## Open it without installing

Download the
[prebuilt example](https://github.com/ankurs/kb/raw/main/examples/wiki.html)
and open it in a browser. It is a single offline file; images are embedded
and non-image source files are omitted.

## Build it yourself

With `kb` [installed](install.md), work on a disposable copy so the
checked-in fixture stays unchanged:

```sh
git clone https://github.com/ankurs/kb.git && cd kb
demo=$(mktemp -d)
rsync -a --exclude '.kb/' --exclude 'dist/' --exclude '.qmd/index.sqlite*' testdata/sample/ "$demo/"
cd "$demo"
kb review
kb find --scope tool-library "seven calendar days"
kb capture --at tool-library "Confirm where returned tools wait for inspection"
kb new meeting --at tool-library --title "Friday Desk Review" --date 2026-10-10
kb check && kb build && kb verify && kb publish
kb serve
```
