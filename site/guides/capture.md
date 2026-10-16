# Capture and review

`kb` keeps capture cheap so that nothing is lost, then gives you tools to turn
rough notes into maintained pages.

## The inbox

```sh
kb capture "Check whether late returns need a fee"
kb capture --at tool-library "Ask about branch returns"
```

Each capture appends one dated line to the `## Unprocessed` section of
`inbox.md`, creating the file or section if needed:

```markdown
## Unprocessed

- 2026-10-09 [tool-library] Ask about branch returns
```

`--at` adds a project label in brackets. Newlines in the text are collapsed so
each capture stays a single list item. Change the file with `capture.inbox`.

## Review

```sh
kb review
```

`kb review` lists the items under `## Unprocessed` without changing the file.
Triage by hand: move each item to another section (for example
`## Triaged`) with a link to where it was resolved.

## Pages from templates

```sh
kb new meeting --at tool-library --title "Desk review" --date 2026-10-10
kb new page --at tool-library --title "Tool repairs"
```

| Kind | Created at | Title line | Template |
| --- | --- | --- | --- |
| `meeting` | `<at>/meetings/YYYY-MM-DD-<slug>.md` | `# YYYY-MM-DD — Title` | `.templates/meeting.md` |
| `page` | `<at>/kb/<slug>.md` | `# Title` | `.templates/kb-page.md` |

`--date` defaults to today. The first line of the template is replaced with
the generated title; the rest is copied as-is. Without a template, the page
contains only its title. `kb new` refuses to overwrite an existing file.
Change the template folder with `capture.templates`.

## A working loop

The [example workspace](../start/example.md) demonstrates a loop that scales well:

1. **Capture** questions and observations as they happen.
2. **Record evidence** in dated, append-only notes such as meetings or a
   journal.
3. **Decide** in a decision log that links to the evidence.
4. **Maintain current pages** (policy, status, actions) that link to the
   decisions behind them.

Because `kb check` validates every link and anchor, the chain from a current
page back to its evidence stays intact as pages are renamed and reorganized.
