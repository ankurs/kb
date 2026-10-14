# Borrowing Pilot — Metrics

> Snapshot at desk close on 2026-10-10. Evidence: [loan ledger extract](../sources/loans-2026-10-10.csv) and [7 October review](../meetings/2026-10-07-pilot-review.md).

## 10 October snapshot

| Measure | Count | Definition |
| --- | ---: | --- |
| Loans opened | 42 | All checkout slips opened from 11 September through 10 October. |
| Due dates reached | 37 | Loans whose seven-day due date was on or before the desk-close cutoff. |
| Returned on time | 35 | Returned by the recorded due date, among the 37 matured loans. |
| Returned late | 2 | Returned after the due date; none remains overdue at this cutoff. |
| Still current | 5 | Due date after 10 October; excluded from the on-time denominator. |
| Day-five reminders | 18 | Reminded matured loans, of which 17 returned on time and one late. |

**On-time rate: 35 / 37 ≈ 94.6%.** This is a descriptive pilot measure, not a causal estimate of reminder effectiveness. The CSV is a small aggregate ledger extract, not personal borrower data.

## Collection and limitations

Eli reconciled desk slips with the [aggregate ledger](../sources/loans-2026-10-10.csv). The five current loans are excluded from return-rate calculations. Two slips were difficult to read before [D-03](../decisions.md#decision-log); both were reconciled against checkout dates. Cross-branch questions are recorded separately and are not loans or returns.

## Aggregate record example

This JSON illustrates the desk-close summary represented by the linked CSV; it contains no borrower identifiers.

```json
{
  "period_end": "2026-10-10",
  "opened": 42,
  "due_reached": 37,
  "returned_on_time": 35,
  "returned_late": 2,
  "still_current": 5
}
```

## Next measurement

In the next cycle, check whether every volunteer reads the due date aloud and whether a return can be completed from the [handbook checklist](../../handbook/check-in/README.md) without a second explanation.
