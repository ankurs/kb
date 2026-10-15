# Mounts

A mount presents a Markdown tree inside the content root under a different
name. Use it to include a shared handbook, a git submodule, or a reference
folder that should also work as a wiki of its own.

```yaml
mounts:
  - name: handbook
    path: reference                         # relative to the content root
    fallback: .kb/snapshots/handbook.html   # optional
```

Pages under `reference/` appear at `handbook/…` in navigation, search, and
every output.

## Links inside a mount

Relative links work as usual. Absolute links such as `/check-in/README.md`
inside a mounted page resolve from the **mount's** root, so the mounted tree
behaves the same whether it is viewed on its own or inside another wiki.

## Fallbacks

If the mount's path is unavailable (for example, an uninitialized submodule),
the HTML bundles written by `kb build` and `kb publish` take the mount's pages
from its `fallback` bundle instead. The live viewer (`kb serve`) and the static
site do not use the fallback: they omit the mount until its path is available
again. Links into the mount are reported as warnings rather than errors.

Build the fallback while the mount is present:

```sh
kb build      # writes .kb/snapshots/handbook.html among the bundles
```

If both the mount and its fallback are missing, the build fails.
