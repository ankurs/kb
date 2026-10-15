# Quick start

This walk-through turns an existing folder of Markdown into a checked,
browsable, and portable wiki. It assumes `kb` is [installed](install.md).

## Create a workspace

A workspace is any directory containing a `kb.yaml`. From your notes folder:

```sh
cd ~/notes
kb init
```

`kb init` writes a minimal `kb.yaml` (`version: 1`) and never overwrites an
existing one. Every other command searches the current directory and its
parents for `kb.yaml`, so you can run `kb` from any subfolder. Use
`--workspace /path/to/notes` to choose one explicitly.

## Check the content

```sh
kb check
```

`kb check` reads every page and reports problems as `page:line: error:
message`. It exits non-zero when it finds errors, so it works in a pre-commit
hook or CI job. It flags:

- pages without exactly one `# H1` title, and duplicate headings on a page;
- links to pages or files that do not exist;
- links to `#anchors` that do not match a heading on the target page;
- links to active files such as HTML or scripts while safe mode is on.

## Browse it

```sh
kb serve
```

Open `http://127.0.0.1:3000`. Navigation is generated from the folder tree,
the page reloads when files change, and Cmd/Ctrl+K searches titles and
content. See [Viewing and building](../guides/viewing.md) for how the sidebar
is ordered and labelled.

## Capture as you work

```sh
kb capture --at projects/garden "Ask about the irrigation schedule"
kb review
kb new meeting --at projects/garden --title "Planning call"
```

Captures land in `inbox.md` under an `## Unprocessed` heading. `kb new` creates
dated meeting notes or topic pages from templates. The
[capture guide](../guides/capture.md) covers the full loop.

## Share it

```sh
kb check && kb publish
```

This writes `dist/wiki.html`, a single file with every page, image, and the
viewer inside it, plus one focused file per top-level folder. Open it directly
from disk or send it to someone. See [Publishing and sync](../guides/publishing.md).

## Next steps

- Learn the vocabulary in [Core concepts](concepts.md).
- Tune navigation, outputs, and publishing in the
  [configuration reference](../reference/configuration.md).
- Explore a finished wiki in the [example workspace](../example.md).
