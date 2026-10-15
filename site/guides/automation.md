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
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.24'}
      - run: go install github.com/ankurs/kb/cmd/kb@latest
      - run: kb check
      - run: kb verify   # only if you commit .kb/snapshots
```

## Hosting the static site

`kb build` writes a static site that any static host can serve: GitHub Pages,
Cloudflare Pages, Netlify, or a plain web server. Upload the contents of
`output.site` (default `.kb/site`).

The site loads its assets from absolute paths, so serve it from the **root**
of a domain or subdomain, such as `docs.example.com`, not from a subpath such
as `example.com/docs/`.

For audiences that do not need a hosted site, attach the `kb publish` output
instead: each HTML file works offline with no server at all.
