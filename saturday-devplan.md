# Saturday Round-Robin + Playoffs — Development Plan

This document details the end-to-end implementation plan for the Saturday special: 16 unique Mon–Fri winners → 4 groups of 4 → 24 round‑robin fights from 10:30 to ~22:00, followed by 3 playoff fights (A vs B, C vs D, Final) at 22:30, 23:00, 23:30. Includes routing, scheduling, generation, repo helpers, UI, background playoff creation, and a code-level kill switch.

## Goals
- New Saturday schedule flow that reuses existing scheduler/engine/websocket.
- Deterministic, idempotent scheduling.
- UI as a normal view (`base.html` top bar), at `/schedule/saturday` with `templates/saturday.html`, `static/css/saturday.css`, `static/js/saturday.js`.
- Code-level kill switch (no env var): `SaturdayRoundRobinEnabled` default `true`. Set `false` to abort.
- Start 10:30, end 23:30 (27 fights).

## High-level Architecture
1) On app start (and each background tick), Saturday schedule is ensured by `EnsureTodaysSchedule`. On Saturdays, branch to `ensureSaturdayRoundRobin` to create 24 group fights only.
2) A background function `MaybeCreateSaturdayPlayoffs` watches for group completion and inserts semifinals and final when inputs are known.
3) UI renders `/schedule/saturday`. If semifinals/final not yet inserted, show placeholders (▓▓▓▓▓▓▓▓▓).

## Router and Handlers
File: `web/server.go`
- Add route:
  - `public.HandleFunc("/schedule/saturday", s.handleSaturday).Methods("GET")`
- In `handleIndex` (after Sunday):
  - If `SaturdayRoundRobinEnabled && now.Weekday()==Saturday` → redirect to `/schedule/saturday`.
- New `handleSaturday(w,r)`:
  - Build `PageData` with `Title: "Saturday Main Event"`, `RequiredCSS: []string{"saturday.css"}`.
  - Get tournament via `scheduler.GetCurrentTournament(now)`.
  - Load today’s fights for display `repo.GetTodaysFights(tournament.ID, today, tomorrow)`.
  - `renderTemplate("saturday.html", data)`; template includes `<script src="/static/js/saturday.js" defer></script>`.

## Kill Switch and Constants
File: `scheduler/scheduler.go`
- Add at top-level:
  - `var SaturdayRoundRobinEnabled = true`
  - `const SaturdayStartHour = 10`
  - `const SaturdayStartMinute = 30`

## Scheduler Flow
File: `scheduler/scheduler.go`
- In `EnsureTodaysSchedule(now)` (before weekday generation):
  - `if now.Weekday()==time.Saturday && SaturdayRoundRobinEnabled { return s.ensureSaturdayRoundRobin(tournament, now) }`

- Implement `ensureSaturdayRoundRobin(t *database.Tournament, now time.Time) error`:
  - Compute Mon–Fri bounds via `utils.GetMonToFriBounds(now)` (Central time).
  - Query completed fights with winners in [Mon00:00, Sat00:00): `repo.GetCompletedFightsInRange(t.ID, start, end)`.
  - Aggregate unique winners, count wins per fighter; sort by wins desc, then by (optional) stat_sum or earliest win time, stable by fighter ID. Take top 16.
  - `generator.GenerateRoundRobinGroups(t, top16, today)` → returns 24 `database.Fight` scheduled 10:30..22:00 every 30m.
  - Insert via `repo.InsertFight` in order.
  - Recovery/activation and `engine.ProcessActiveFights(now)` as already done for normal days.
  - Do NOT create semifinals/final here.

- Background playoff creation (in main.go ticker):
  - Call `sched.MaybeCreateSaturdayPlayoffs(now)` each tick (only on Saturday; guarded by flag).

- Implement `MaybeCreateSaturdayPlayoffs(now time.Time) error`:
  - Get tournament; get today/tomorrow via `utils.GetDayBounds(now)`.
  - Fetch all today fights once `repo.GetTodaysFights` and partition by fixed time windows:
    - Group A: 10:30–13:00 (6 fights)
    - Group B: 13:30–16:00 (6 fights)
    - Group C: 16:30–19:00 (6 fights)
    - Group D: 19:30–22:00 (6 fights)
    - Playoffs times: SF1 22:30, SF2 23:00, Final 23:30
  - If all A and B fights `status='completed'` and no fight exists at 22:30, compute each group’s winner:
    - Sort by wins (count of group fights won), tiebreak by cumulative (final_score_for - final_score_against) across group, then earliest `completed_at`.
    - Insert SF1 at 22:30 with the two winners.
  - If all C and D completed and no 23:00 fight, compute winners and insert SF2 at 23:00.
  - If both semis completed and no 23:30 fight, read their winners and insert Final at 23:30.
  - Idempotence: use `repo.FightExistsAt(tournamentID, exactTime)`; if true, skip.

## Generator Additions
File: `fight/generator.go`
- Add helper:
  - `func (g *Generator) GenerateRoundRobinGroups(t *database.Tournament, entrants []database.Fighter, date time.Time) ([]database.Fight, error)`
    - Expect `len(entrants)>=16`; if more, take top 16; if fewer, degrade (12→3 groups; 8→2 groups) and still fill 24 slots with show-cases or compact schedule (stretch goal: not required for v1 if we can guarantee 16).
    - Seed into groups A–D deterministically (wins desc; stable by ID).
    - For each group, produce matches in order: (1–2,3–4),(1–3,2–4),(1–4,2–3) → 6 fights.
    - Schedule slots starting at `SaturdayStartHour:SaturdayStartMinute` at 30m increments: A(0..5), B(6..11), C(12..17), D(18..23).
    - Return 24 fights.

## Repository Helpers
File: `database/repository.go`
- Add:
  - `func (r *Repository) GetCompletedFightsInRange(tID int, start, end time.Time) ([]Fight, error)`
    - SQL: `SELECT * FROM fights WHERE tournament_id=? AND status='completed' AND winner_id IS NOT NULL AND completed_at>=? AND completed_at<? ORDER BY completed_at ASC`
  - `func (r *Repository) FightExistsAt(tID int, at time.Time) (bool, error)`
    - SQL: `SELECT COUNT(*) FROM fights WHERE tournament_id=? AND scheduled_time=?`

## Utilities
File: `utils/time.go`
- Add `GetMonToFriBounds(now time.Time) (time.Time, time.Time)` using Central time; return Monday 00:00 of current week to Saturday 00:00.

## UI
- Template: `templates/saturday.html` (Go template; uses `{{template "content" .}}` with `base.html`)
  - Layout matching the mock: four group columns (2×2 grid), standings live, a Live Now card + Next Fight countdown, and a “Playoffs” section showing SF/Final with redacted names until inserted.
- CSS: `static/css/saturday.css` (lift styles from mock; variables from base.css)
- JS: `static/js/saturday.js`
  - Builds timeline from fights in DOM/JSON payload; marks completed/live/upcoming; left-edge color based on whether the user has a pending bet; redacts playoff names when fights don’t exist yet.
  - Optional: poll every 30–60s for new fights (semis/final) to reveal names.

## Data Flow
1) Saturday morning:
   - `EnsureTodaysSchedule` → `ensureSaturdayRoundRobin` inserts 24 fights.
2) All day:
   - Background ticker every 30s calls `MaybeCreateSaturdayPlayoffs` to insert SF/Final when ready.
3) UI:
   - Renders fights immediately; placeholders for playoffs until rows exist.

## Idempotence and Safety
- Generation only runs if `GetTodaysFights` returns 0 rows.
- Playoff creation checks exact `scheduled_time` existence before inserting.
- Kill switch: set `SaturdayRoundRobinEnabled=false` and restart → normal Saturday schedule resumes (no redirect, no special generation).

## Testing Plan
1) Seed DB with fighters and a week’s tournament; mark a subset of Mon–Fri fights completed with winners.
2) Run on a Saturday date in Central; verify 24 fights inserted at exact slots.
3) Mark A+B groups completed; tick once → verify SF1 inserted at 22:30.
4) Mark C+D completed; tick once → verify SF2 inserted at 23:00.
5) Resolve both semis; tick once → verify Final at 23:30.
6) UI verifies placeholders until rows exist; then real names appear.

## Files to Change / Add (with anchors)
- `web/server.go`
  - Add Saturday route in `setupRoutes` (132–163).
  - Redirect in `handleIndex` (241–316).
  - New `handleSaturday`.
- `scheduler/scheduler.go`
  - Add flag/constants (top of file).
  - Saturday branch in `EnsureTodaysSchedule` (61–165).
  - Implement `ensureSaturdayRoundRobin`, `MaybeCreateSaturdayPlayoffs`.
- `fight/generator.go`
  - Add `GenerateRoundRobinGroups`.
- `database/repository.go`
  - Add `GetCompletedFightsInRange`, `FightExistsAt`.
- `utils/time.go`
  - Add `GetMonToFriBounds`.
- Add assets
  - `templates/saturday.html`
  - `static/css/saturday.css`
  - `static/js/saturday.js`

## Open Questions (post‑v1)
- Tie-breaking depth (stat_sum vs. ELO-like rating). For v1, do wins → point diff → earliest completion.
- If unique winners < 16: degrade to fewer groups + showcases or pull top non-winners by stat sum.
- Optional: mark Saturday fights with a tag (no schema change for v1; time windows are sufficient for identification).


