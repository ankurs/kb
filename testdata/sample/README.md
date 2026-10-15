# Neighborhood Tool Library

A local, fictional knowledge base for a 30-day borrowing pilot, **11 September–10 October 2026**. Seven volunteers use it to record what happened, reconcile policy, and prepare desk handoffs. All people, counts, and rules here are invented for this example.

## Start here

- [Now](NOW.md) — current priorities and the next desk shift.
- [Inbox](inbox.md) — uncategorized questions captured during work.
- [Tool Library Pilot](tool-library/README.md) — current state, policy, decisions, actions, and dated evidence.
- [Volunteer Handbook](handbook/README.md) — a mounted, reusable desk procedure with a diagram.

## Workspace configuration

The checked-in [kb.yaml](kb.yaml) makes this directory a `kb` workspace. Its
`version`, `id`, and `name` identify the example and label its viewers. The
`handbook` mount presents Markdown from `reference/` under `handbook/` in the
wiki. If that directory is temporarily unavailable, a previously built
`.kb/snapshots/handbook.html` supplies its pages for builds and publishing;
build that snapshot while the mount is present before relying on the fallback.

`navigation.overrides` labels the pilot section **Tool Library Pilot** and
places it first. `render.trust_html: false` explicitly keeps the default safe
rendering behavior: the viewers sanitize Markdown HTML, Mermaid runs in strict
mode, active linked files are rejected, and unsafe SVG assets are rejected. Do
not enable `trust_html` for notes from untrusted authors.

`output` places the generated site in `.kb/site/` and offline HTML snapshots
in `.kb/snapshots/`. Separately, `publish` writes `dist/wiki.html` plus focused
HTML files, renaming the `tool-library` bundle to `borrowing-pilot.html`.
`sync.command` copies an **already published** `dist/` into the local
`.kb/mirror/`; neither `kb build` nor `kb publish` runs sync automatically.
The remaining settings use built-in defaults, including the local-only server,
`inbox.md` and `.templates/` for capture, and `.qmd/` for optional qmd search.

## Feature map

| Capability | Where to see it | Try it |
| --- | --- | --- |
| Dated capture and triage | [Inbox](inbox.md) and [30-day journal](tool-library/journal/README.md) | `kb review`; `kb capture --at tool-library "Follow up"` |
| Page creation | [Meeting template](.templates/meeting.md) and [KB template](.templates/kb-page.md) | `kb new meeting --at tool-library --title "Desk review"`; `kb new page --at tool-library --title "Tool repairs"` |
| Evidence to current knowledge | [Posted notice](tool-library/sources/desk-rules.txt) → [11 Sep meeting](tool-library/meetings/2026-09-11-kickoff.md) → [D-01](tool-library/decisions.md) → [policy](tool-library/kb/borrowing-policy.md) | `kb check` |
| Keyword and hybrid retrieval | [qmd collections](.qmd/index.yml) and [retrieval checks](.qmd/bench-tool-library.json) | `kb find --scope tool-library "seven calendar days"`; `kb search --scope wiki "reminder timing"` after indexing |
| Rendered Mermaid and code | [Reminder flow and YAML](tool-library/kb/reminder-workflow.md#flow), [metrics JSON](tool-library/kb/pilot-metrics.md#aggregate-record-example), and [shift handoff](handbook/shifts/README.md#handoff-flow) | `kb serve --port 3001` |
| Local image | [Check-in SVG](handbook/check-in/check-in-flow.svg) | `kb build`; open `.kb/snapshots/wiki.html` offline |
| Mounted reference notes | [Volunteer Handbook](handbook/README.md), sourced from `reference/` | `kb serve`; `kb search --scope handbook "check in a tool"` |
| Source links and metrics | [Ledger CSV](tool-library/sources/loans-2026-10-10.csv) → [metrics page](tool-library/kb/pilot-metrics.md) | `kb check`; `kb publish` |
| Independent commands | [Local output configuration](kb.yaml) | `kb build`; `kb publish`; `kb sync --show`; `kb sync` |
| Diagnostics | [Workspace configuration](kb.yaml) | `kb doctor`; `kb -v check`; `kb verify` |

## A normal day

1. Record a question: `kb capture --at tool-library "Ask how to record a late return"`.
2. Locate evidence: `kb find --scope tool-library "reminder"` or `kb search --scope wiki "when do reminders go out"`.
3. Make a dated page with `kb new meeting --at tool-library --title "Desk review" --date 2026-10-10` or a current page with `kb new page --at tool-library --title "Tool repairs"`. Read the [source register](tool-library/sources/README.md), a [dated meeting](tool-library/meetings/2026-09-22-reminder-trial.md), then the [current policy](tool-library/kb/borrowing-policy.md). The journal records every day of the pilot in [five weekly files](tool-library/journal/README.md).
4. Update the [actions](tool-library/actions.md) and [status](tool-library/status.md) after a shift; keep meeting notes as historical evidence.
5. Run `kb check` before `kb build` or `kb publish`. `kb serve --port 3001` gives a live browser view.

## Retrieval and offline copies

The checked-in [qmd configuration](.qmd/index.yml) separates `wiki` from `handbook`; it contains no search database. Run `qmd init` once in this directory to create `.qmd/index.sqlite`, then `kb index` and `kb search --scope wiki "seven-day borrowing policy"`. Try `kb search --scope handbook "inspect a returned tool"` for the mounted reference collection or `qmd bench .qmd/bench-tool-library.json -c wiki` for [retrieval checks](.qmd/bench-tool-library.json). `kb find` always works without qmd or downloaded models. Search results point to documents; they do not synthesize an answer.

`kb build` makes a local site and offline snapshots in `.kb/`. `kb publish` writes `dist/wiki.html`, `dist/borrowing-pilot.html`, and `dist/handbook.html`. `kb sync --show` previews the configured command; `kb sync` copies the published files into `.kb/mirror/` on this machine. No network or external destination is involved. The publish step includes Markdown and images, but not the linked raw source files.
