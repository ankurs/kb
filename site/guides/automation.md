# Automating checks

## Pre-commit hook

```sh
cat > .git/hooks/pre-commit <<'HOOK'
#!/bin/sh
exec kb check
HOOK
chmod +x .git/hooks/pre-commit
```

## Continuous integration

A GitHub Actions job that validates a notes repository on every push:

```yaml
name: Notes
on: [push, pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with: {go-version: stable}
      - run: go install github.com/ankurs/kb/cmd/kb@latest
      - run: kb check
      - run: kb verify   # only if you commit both .kb/site and .kb/snapshots
```

`kb verify` compares committed output with what `kb build` would write now, so
do not run `kb build` before it in CI: that would regenerate the output and
hide a stale commit.

## Hosting the static site

`kb build` writes a static site that any static host can serve: GitHub Pages,
Cloudflare Pages, Netlify, or a plain web server. Upload the contents of
`output.site` (default `.kb/site`).

The site uses relative paths, so it works at the root of a domain, such as
`docs.example.com`, or under a path, such as a GitHub Pages project site at
`example.github.io/notes/`.

For audiences that do not need a hosted site, attach the `kb publish` output
instead: each HTML file works offline with no server at all.
