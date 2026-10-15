# Search

`kb` offers three kinds of search, from zero-setup to semantic.

## Keyword search: `kb find`

```sh
kb find "seven calendar days"
kb find --scope tool-library "reminder"
```

Searches Markdown directly, with no index, model, or network access.
`--scope` limits results to a path prefix.

## Browser search

Every viewer, including the single-file bundles, has ranked search over page
titles and content. Press Cmd/Ctrl+K to focus it and Escape to return to the
navigation. It works offline and needs no setup.

## Hybrid retrieval with qmd

For semantic and hybrid retrieval, `kb` delegates to
[qmd](https://github.com/tobi/qmd), a local search engine for Markdown. It is
optional; install it separately.

```sh
qmd init                                   # create .qmd/index.sqlite
kb index                                   # qmd update + qmd embed
kb search --scope wiki "when is the day-five reminder sent?"
```

- `kb index` updates documents and generates embeddings. Models download
  once; `--force` re-embeds everything.
- `kb search` returns ranked documents and snippets, not a synthesized answer.
  `--scope` selects a qmd collection.
- `search.qmd_config` (default `.qmd`) is where the project's qmd
  configuration lives. Define collections in `.qmd/index.yml`.

`kb doctor` reports a missing project configuration or empty collections. A
qmd configuration with update hooks, collection paths outside the project, or
custom models must be approved with `qmd trust`; `kb` never changes trust
settings for you.
