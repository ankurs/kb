# Publishing and sync

## Publish a distribution

```sh
kb publish
```

`kb publish` writes an HTML-only distribution to `publish.dir` (default
`dist/`):

- `wiki.html` with every selected page;
- one focused HTML file per top-level folder, named after the folder.

Markdown and referenced images are embedded. Non-image files such as CSV or
PDF are **not** copied, so links to them are inactive in published HTML.
`kb publish` does not run `kb build` or `kb sync`.

## Choose what to publish

```yaml
publish:
  dir: dist
  wiki: wiki.html
  include: [handbook, guides]   # only these top-level folders
  exclude: [drafts]             # or: everything except these
  root: true                    # keep root pages when filtering
  names:
    guides: team-guides.html    # rename a focused file
```

With neither `include` nor `exclude`, every folder and root page is
published. When either is set, root pages are omitted unless `root: true`.
Selection applies to the combined wiki and the focused files alike.
Publishing reads only the selected pages and what they link to, so an
unavailable mount that is excluded does not block the rest. It does not run
the heading, link, and anchor checks of `kb check`, and links to missing
local files are simply left inactive; run `kb check` before publishing.

One-off overrides on the command line:

```sh
kb publish --output /tmp/share --include handbook --root-pages
kb publish --exclude drafts,private --wiki everything.html
```

## Safe replacement

`kb publish` marks its output directory with a `.kb-published` file and only
ever replaces a directory carrying that marker. If `dist/` already holds
something else, move it aside first. `publish.dir` may be relative to the
workspace or an absolute path outside it.

## Sync

```yaml
sync:
  command: >-
    rsync -av -- "$KB_PUBLISH_DIR/" "$HOME/wiki-backup/"
```

```sh
kb sync --show   # print the working directory, publish dir, and command
kb sync          # run it
```

`kb sync` runs `sync.command` with `sh -c` from the workspace root and sets
`KB_WORKSPACE_ROOT` and `KB_PUBLISH_DIR` as absolute paths. It does not
publish first; run `kb publish && kb sync`. `--show` prints the command
without running it; it does not simulate its effects.

> `sync.command` is arbitrary shell. Treat `kb.yaml` from someone else like a
> script: read it before running `kb sync`.
