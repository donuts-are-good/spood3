#!/usr/bin/env bash
set -euo pipefail

# CONFIG
WIKI_BASE="https://spoodblort.fandom.com"
API="$WIKI_BASE/api.php"
DB="spoodblort.db"   # absolute path for reliability
BOT_USER="SpoodblortCommissioner@Spoodbot"     # BotPassword login name (YourUser@Label)
# Hard-code your Bot Password (keep this script private)
BOT_PASS="q15iu06t2dk5adia5r9b37r4ibt5rvm1"

# deps: jq, sqlite3, curl
command -v jq >/dev/null || { echo "Install jq"; exit 1; }
command -v sqlite3 >/dev/null || { echo "Install sqlite3"; exit 1; }
command -v curl >/dev/null || { echo "Install curl"; exit 1; }

COOKIE="$(mktemp)"
cleanup() { rm -f "$COOKIE"; }
trap cleanup EXIT

# Options
ONLY_STATUS="${ONLY_STATUS:-all}"   # scheduled|active|completed|voided|all
DRY_RUN=0       # 1 to preview titles only
RATE_MS=10000     # throttle between edits (ms)

# Persistent queue + scheduling
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
QUEUE_FILE="$SCRIPT_DIR/.create_fights.queue"
SLEEP_SECONDS=$((12*60*60))

perform_login() {
  echo "[1/4] Get login token"
  LOGIN_TOKEN=$(curl -s "$API?action=query&meta=tokens&type=login&format=json" -c "$COOKIE" | jq -r '.query.tokens.logintoken')

  echo "[2/4] Login"
  LOGIN_RESULT=$(curl -s "$API?action=login&format=json" -b "$COOKIE" -c "$COOKIE" \
    --data-urlencode "lgname=$BOT_USER" \
    --data-urlencode "lgpassword=$BOT_PASS" \
    --data-urlencode "lgtoken=$LOGIN_TOKEN" | jq -r '.login.result')
  if [[ "$LOGIN_RESULT" != "Success" ]]; then
    echo "Login failed: $LOGIN_RESULT" >&2
    return 1
  fi

  echo "[3/4] Get CSRF token"
  CSRF=$(curl -s "$API?action=query&meta=tokens&type=csrf&format=json" -b "$COOKIE" | jq -r '.query.tokens.csrftoken')
}

# Ensure Template:Fight exists
ensure_template() {
  local exists
  exists=$(curl -s "$API?action=query&format=json&titles=Template:Fight" -b "$COOKIE" | jq -r '.query.pages|to_entries[0].value.missing // "0"')
  if [[ "$exists" == "0" ]]; then
    echo "[Template] Template:Fight exists"
    return
  fi
  echo "[Template] Creating Template:Fight"
  TEMPLATE_TEXT='{| class="infobox"
! colspan="2" style="text-align:center;" | {{PAGENAME}}
|-
! Tournament
| {{{tournament|}}}
|-
! Scheduled
| {{{scheduled|}}}
|-
! Fighters
| {{{fighter1|}}} vs {{{fighter2|}}}
|-
! Status
| {{{status|}}}
|-
! Result
| Winner: {{{winner|}}}
|}'
  curl -s "$API?action=edit&format=json" -b "$COOKIE" \
    --data-urlencode "title=Template:Fight" \
    --data-urlencode "text=$TEMPLATE_TEXT" \
    --data-urlencode "summary=Add Fight template" \
    --data-urlencode "token=$CSRF" \
    --data-urlencode "createonly=1" | jq -r '.edit.result'
}

build_sql() {
  SQL='SELECT f.id, f.fighter1_id, f.fighter2_id, f.fighter1_name, f.fighter2_name, f.scheduled_time, f.status, f.winner_id, f.final_score1, f.final_score2,\
               COALESCE(t.name, "Unknown Tournament") AS tournament_name\
        FROM fights f LEFT JOIN tournaments t ON f.tournament_id = t.id'
  if [[ "$ONLY_STATUS" != "all" ]]; then
    SQL+=" WHERE f.status = '"$ONLY_STATUS"'"
  fi
  SQL+=' ORDER BY f.id;'
}

refresh_queue_if_needed() {
  if [[ ! -s "$QUEUE_FILE" ]]; then
    echo "[Queue] Building queue file at $QUEUE_FILE"
    build_sql
    echo "[4/4] Creating/Updating fight pages"
    echo "[DEBUG] SQL Query: $SQL"
    sqlite3 -json "$DB" "$SQL" | jq -c '.[]' > "$QUEUE_FILE"
  else
    echo "[Queue] Resuming from existing queue: $QUEUE_FILE"
  fi
}

process_queue_cycle() {
  while true; do
    row=$(head -n 1 "$QUEUE_FILE" || true)
    if [[ -z "$row" ]]; then
      echo "[Queue] Completed all entries"
      break
    fi

    id=$(jq -r '.id' <<<"$row")
    f1=$(jq -r '.fighter1_name' <<<"$row")
    f2=$(jq -r '.fighter2_name' <<<"$row")
    scheduled=$(jq -r '.scheduled_time' <<<"$row")
    status=$(jq -r '.status' <<<"$row")
    tournament=$(jq -r '.tournament_name' <<<"$row")
    winner_id=$(jq -r '.winner_id // empty' <<<"$row")
    f1_id=$(jq -r '.fighter1_id' <<<"$row")
    f2_id=$(jq -r '.fighter2_id' <<<"$row")
    score1=$(jq -r '.final_score1 // empty' <<<"$row")
    score2=$(jq -r '.final_score2 // empty' <<<"$row")

    if [[ -n "$winner_id" && "$winner_id" != "null" ]]; then
      if [[ "$winner_id" == "$f1_id" ]]; then
        winner="$f1"
      elif [[ "$winner_id" == "$f2_id" ]]; then
        winner="$f2"
      else
        winner=""
      fi
    else
      winner=""
    fi

    DISPLAY_TITLE=$(printf "Fight %d %s vs %s" "$id" "$f1" "$f2")
    SAFE_F1=$(printf '%s' "$f1" | sed 's/[^A-Za-z0-9 _-]/_/g' | tr ' ' '_')
    SAFE_F2=$(printf '%s' "$f2" | sed 's/[^A-Za-z0-9 _-]/_/g' | tr ' ' '_')
    TITLE=$(printf "Fight_%d_%s_vs_%s" "$id" "$SAFE_F1" "$SAFE_F2")

    if [[ "$DRY_RUN" == "1" ]]; then
      echo "[DRY] Would create: $TITLE"
      tail -n +2 "$QUEUE_FILE" > "$QUEUE_FILE.tmp" && mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"
      continue
    fi

    result_section="To be determined."
    if [[ -n "$winner" ]]; then
      if [[ -n "$score1" && -n "$score2" && "$score1" != "null" && "$score2" != "null" ]]; then
        result_section="Winner: ${winner} — Final Score ${score1}-${score2}."
      else
        result_section="Winner: ${winner}."
      fi
    fi

    TEXT=$(cat <<EOF
{{DISPLAYTITLE:$DISPLAY_TITLE}}
{{Fight
|tournament=$tournament
|scheduled=$scheduled
|fighter1=$f1
|fighter2=$f2
|status=$status
|winner=$winner
}}

'''$DISPLAY_TITLE'''

== Overview ==
This bout is scheduled under the '''$tournament''' card.

== Result ==
$result_section

[[Category:Fights]]
EOF
    )

    retries=0
    edit_success=0
    while true; do
      resp=$(curl -s "$API?action=edit&format=json" -b "$COOKIE" \
        --data-urlencode "title=$TITLE" \
        --data-urlencode "text=$TEXT" \
        --data-urlencode "summary=Sync fight page from game DB" \
        --data-urlencode "token=$CSRF")

      result=$(jq -r '.edit.result // empty' <<<"$resp")
      if [[ "$result" == "Success" ]]; then
        echo "✅ Synced: $TITLE"
        edit_success=1
        break
      fi

      errcode=$(jq -r '.error.code // empty' <<<"$resp")
      if [[ "$errcode" == "ratelimited" ]]; then
        retries=$((retries + 1))
        echo "⚠️  Rate limited while syncing $TITLE. Retry #$retries"
        sleep 5
        RATE_MS=$((RATE_MS + 1000))
        continue
      fi

      echo "⚠️  Error for $TITLE: $resp"
      break
    done

    if [[ "$edit_success" == "1" ]]; then
      tail -n +2 "$QUEUE_FILE" > "$QUEUE_FILE.tmp" && mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"
    else
      echo "[Queue] Keeping entry for retry: $TITLE"
    fi

    if [[ "$RATE_MS" -gt 0 ]]; then
      sleep "$(awk -v ms="$RATE_MS" 'BEGIN{printf "%.3f", ms/1000}')"
    fi
  done
}

while true; do
  if ! perform_login; then
    echo "[Auth] Login failed; sleeping 60s before retry"
    sleep 60
    continue
  fi
  ensure_template
  refresh_queue_if_needed
  echo "[DEBUG] Processing $(wc -l < "$QUEUE_FILE") fights"
  process_queue_cycle
  rm -f "$QUEUE_FILE"
  echo "[Scheduler] Cycle complete. Sleeping for $SLEEP_SECONDS seconds (12h)"
  sleep "$SLEEP_SECONDS"
done
