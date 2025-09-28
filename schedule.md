## Schedule API (/api/schedule/today)

### Overview
- **Method**: GET
- **Auth**: none
- **Path**: `/api/schedule/today`
- **Status**: always 200 OK (errors are conveyed in the JSON body)
- **Content-Type**: `application/json; charset=utf-8`

### Request

```bash
curl -s http://localhost:8080/api/schedule/today
```

### Response shape

- **meta**
  - **now**: RFC3339 timestamp in America/Chicago
  - **day**: YYYY-MM-DD (America/Chicago)
  - **timezone**: string (always "America/Chicago")
  - **tournament_id**: integer, optional (present when a current tournament exists)
- **fights**: array of fights for today (America/Chicago day bounds)
  - **id**: integer
  - **tournament_id**: integer
  - **fighter1_id**: integer
  - **fighter2_id**: integer
  - **fighter1_name**: string
  - **fighter2_name**: string
  - **scheduled_time**: RFC3339 timestamp in America/Chicago
  - **status**: string (e.g., "scheduled", "active", "completed")
  - **winner_id**: integer, optional
  - **final_score1**: integer, optional
  - **final_score2**: integer, optional
  - **completed_at**: RFC3339 timestamp, optional
- **error**: string, optional (set on failures)

Notes
- Timestamps are returned in RFC3339 with the proper -05:00/-06:00 offset depending on DST.
- Optional fields are omitted when empty.

### Example responses

Success with fights (current tournament found)

```json
{
  "meta": {
    "now": "2025-09-28T11:07:00-05:00",
    "day": "2025-09-28",
    "timezone": "America/Chicago",
    "tournament_id": 42
  },
  "fights": [
    {
      "id": 101,
      "tournament_id": 42,
      "fighter1_id": 7,
      "fighter2_id": 12,
      "fighter1_name": "Glizzy Goblin",
      "fighter2_name": "Moon Landing Mike",
      "scheduled_time": "2025-09-28T13:00:00-05:00",
      "status": "scheduled"
    },
    {
      "id": 102,
      "tournament_id": 42,
      "fighter1_id": 9,
      "fighter2_id": 3,
      "fighter1_name": "War Machine",
      "fighter2_name": "Average Joe",
      "scheduled_time": "2025-09-28T10:30:00-05:00",
      "status": "completed",
      "winner_id": 9,
      "final_score1": 512,
      "final_score2": 300,
      "completed_at": "2025-09-28T10:45:12-05:00"
    }
  ]
}
```

Success but no current tournament (empty list; no `tournament_id`)

```json
{
  "meta": {
    "now": "2025-09-28T11:07:00-05:00",
    "day": "2025-09-28",
    "timezone": "America/Chicago"
  },
  "fights": []
}
```

Repo error fetching fights (has `tournament_id`, empty fights, `error` set)

```json
{
  "meta": {
    "now": "2025-09-28T11:07:00-05:00",
    "day": "2025-09-28",
    "timezone": "America/Chicago",
    "tournament_id": 42
  },
  "fights": [],
  "error": "database timeout"
}
```

Scheduler error (no `tournament_id`, empty fights, `error` set)

```json
{
  "meta": {
    "now": "2025-09-28T11:07:00-05:00",
    "day": "2025-09-28",
    "timezone": "America/Chicago"
  },
  "fights": [],
  "error": "no schedule available"
}
```


