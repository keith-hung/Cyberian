---
name: wedaka
description: "Clock in/out and manage attendance in WeDaka. Trigger when user asks about: clock in, clock out, punch in, punch out, attendance, timelog."
user-invokable: true
argument-hint: "[action, e.g. 'clock-in', 'clock-out', 'timelog']"
---

# WeDaka Skill

Manage employee clock-in/out records and attendance via the `wedaka` CLI.

## Trigger

Activate when user asks to:
- Clock in or out (打卡、上班、下班)
- Check attendance or time log records (考勤、出勤紀錄)
- Verify if a date is a work day (工作日查詢)

## Prerequisites

Environment variables must be set before use:
- `WEDAKA_API_URL` — WeDaka API base URL
- `WEDAKA_USERNAME` — Employee username
- `WEDAKA_EMP_NO` — Employee number
- `WEDAKA_DEVICE_ID` — Device UUID for authentication

The CLI binary is in `${CLAUDE_PLUGIN_ROOT}/scripts/`. Invoke via the launcher:
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh <command> [flags]
```

## Commands

### Clock in
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh clock-in [--date YYYY-MM-DD] [--time HH:MM:SS] [--note "text"]
```
Records start of work. Date defaults to today, time defaults to now.

### Clock out
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh clock-out [--date YYYY-MM-DD] [--time HH:MM:SS] [--note "text"]
```
Records end of work. Same defaults as clock-in.

### Query time log
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh timelog --month <1-12> --year <YYYY>
```
Returns all clock-in/out and leave records for the specified month.
Output includes `summary` (clock_ins, clock_outs, leaves counts) and `records[]`.

### Check work day
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh check-workday --date YYYY-MM-DD
```
Returns whether the date is a work day, leave day, or holiday.

### Version
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh version
```

## Important Rules

- Cannot clock for future dates — only today or past dates
- Cannot clock on non-work days (holidays, leave days) — the CLI checks automatically
- Date format must be YYYY-MM-DD, time format must be HH:MM:SS
- Always confirm the current date/time with the user before clocking

## Workflow

1. **Confirm** date and time with user
2. **Check** if the date is a work day (optional — clock-in/out checks automatically)
3. **Clock** in or out
4. **Verify** by querying the time log for the current month

## Error Handling

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | — |
| 1 | General error | Check error message |
| 2 | Config error | Verify environment variables |
| 3 | Validation error | Check input format or date constraints |
| 4 | Network error | Verify API URL is reachable |

Errors are JSON on stderr: `{"success":false,"error":"message"}`.

## Example: Clock In for Today

```bash
# 1. Clock in
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh clock-in --time 09:00:00

# 2. Verify
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh timelog --month 3 --year 2026
```

## Example: Retroactive Clock Out

```bash
# Clock out for a past date
${CLAUDE_PLUGIN_ROOT}/scripts/wedaka-launcher.sh clock-out --date 2026-02-28 --time 18:00:00 --note "Forgot to clock out"
```
