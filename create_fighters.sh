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
# Lore formatting relies on Python; optional. Set to empty to fall back to shell.
PYTHON_BIN="$(command -v python3 || command -v python || true)"
if [[ -z "$PYTHON_BIN" ]]; then
  echo "[Lore] python3 not found; using shell formatter"
fi

COOKIE="$(mktemp)"
cleanup() { rm -f "$COOKIE"; }
trap cleanup EXIT

# Options (hard-coded)
ONLY_ALIVE=0    # 1 to skip dead fighters
DRY_RUN=0       # 1 to preview titles only
RATE_MS=2000     # throttle between edits (ms)
INDEX_UPDATED=0  # Track if we've attempted index update to avoid rate limiting

echo "[1/4] Get login token"
LOGIN_TOKEN=$(curl -s "$API?action=query&meta=tokens&type=login&format=json" -c "$COOKIE" | jq -r '.query.tokens.logintoken')

echo "[2/4] Login"
LOGIN_RESULT=$(curl -s "$API?action=login&format=json" -b "$COOKIE" -c "$COOKIE" \
  --data-urlencode "lgname=$BOT_USER" \
  --data-urlencode "lgpassword=$BOT_PASS" \
  --data-urlencode "lgtoken=$LOGIN_TOKEN" | jq -r '.login.result')
if [[ "$LOGIN_RESULT" != "Success" ]]; then
  echo "Login failed: $LOGIN_RESULT" >&2
  exit 1
fi

echo "[3/4] Get CSRF token"
CSRF=$(curl -s "$API?action=query&meta=tokens&type=csrf&format=json" -b "$COOKIE" | jq -r '.query.tokens.csrftoken')

# Optional: ensure Template:Fighter exists (create only if missing)
ensure_template() {
  local exists
  exists=$(curl -s "$API?action=query&format=json&titles=Template:Fighter" -b "$COOKIE" | jq -r '.query.pages|to_entries[0].value.missing // "0"')
  if [[ "$exists" == "0" ]]; then
    echo "[Template] Template:Fighter exists"
    return
  fi
  echo "[Template] Creating Template:Fighter"
  TEMPLATE_TEXT='{| class="infobox"
! colspan="2" style="text-align:center;" | {{PAGENAME}}
|-
! Team
| {{{team|}}}
|-
! Class
| {{{class|}}}
|-
! Stats
| Str: {{{strength|}}} • Spd: {{{speed|}}} • End: {{{endurance|}}} • Tec: {{{technique|}}}
|-
! Lore
| {{{lore|}}}
|}
[[Category:Fighters]]'
  curl -s "$API?action=edit&format=json" -b "$COOKIE" \
    --data-urlencode "title=Template:Fighter" \
    --data-urlencode "text=$TEMPLATE_TEXT" \
    --data-urlencode "summary=Add Fighter template" \
    --data-urlencode "token=$CSRF" \
    --data-urlencode "createonly=1" | jq -r '.edit.result'
}
ensure_template

# Update Fighters index page: insert a bullet for the new fighter in numeric order
add_to_fighters_index() {
  local title="$1"
  local display="$2"

  local content
  content=$(curl -sL -H "User-Agent: SpoodblortBot/1.0" -b "$COOKIE" "$WIKI_BASE/wiki/Fighters?action=raw&ctype=text/plain")
  if [[ -z "$content" || "$content" == "null" ]]; then
    echo "[Index] Raw fetch empty, using parse API"
    content=$(curl -s "$API?action=parse&page=Fighters&prop=wikitext&formatversion=2&format=json" -b "$COOKIE" | jq -r '.parse.wikitext // ""')
    if [[ -z "$content" ]]; then
      echo "[Index] Could not load Fighters page; skipping index update"
      return 0
    fi
  fi

  if grep -Fq "[[${display}]]" <(printf '%s' "$content") || grep -Fq "[[${title}]]" <(printf '%s' "$content"); then
    echo "[Index] Already listed: $display"
    return 0
  fi

  local tmp
  tmp=$(mktemp)
  printf '%s' "$content" > "$tmp"

  local start end
  start=$(awk 'BEGIN{IGNORECASE=1} /^Notable[[:space:]]+Fighters:/{print NR; exit}' "$tmp") || true
  if [[ -z "$start" ]]; then
    echo "[Index] Could not find 'Notable Fighters:' heading; skipping index update"
    rm -f "$tmp"
    return 0
  fi

  end=$(awk -v s="$start" 'NR>s { if ($0=="" || $1!="*") { print NR; exit } } END{if(!NR)print 0}' "$tmp") || true
  if [[ -z "$end" || "$end" == 0 ]]; then
    end=$(wc -l < "$tmp")
  fi

  local pre block post bullets others newcontent
  pre=$(awk -v e="$start" 'NR<=e{print}' "$tmp")
  block=$(awk -v s="$start" -v e="$end" 'NR>s && NR<e{print}' "$tmp")
  post=$(awk -v e="$end" 'NR>=e{print}' "$tmp")

  bullets=$(printf '%s\n' "$block" | awk '/^\*\s*Roster\s*#/{print}')
  others=$(printf '%s\n' "$block" | awk '!/^\*\s*Roster\s*#/{print}')

  bullets=$(printf '%s\n\n* [[%s|%s]]\n' "$bullets" "$title" "$display" | awk 'NF')
  bullets=$(printf '%s\n' "$bullets" | sort -t# -k2,2n)

  newcontent=$(printf '%s\n%s\n%s' "$pre" "$bullets" "$others$post")

  resp=$(curl -s "$API?action=edit&format=json" -b "$COOKIE" \
    --data-urlencode "title=Fighters" \
    --data-urlencode "text=$newcontent" \
    --data-urlencode "summary=Add $display to Fighters index" \
    --data-urlencode "token=$CSRF")
  if [[ $(echo "$resp" | jq -r '.edit.result // empty') == "Success" ]]; then
    echo "[Index] Inserted into Fighters: $display"
    rm -f "$tmp"
    return 0
  fi

  local err
  err=$(echo "$resp" | jq -r '.error.code // empty')
  if [[ "$err" == "ratelimited" ]]; then
    echo "[Index] Rate limited while updating Fighters. Backing off..."
    rm -f "$tmp"
    return 2
  fi

  echo "[Index] Edit failed for Fighters: $resp"
  rm -f "$tmp"
  return 0
}

# Build SQL
SQL='SELECT id, name, team, fighter_class AS class,
             strength, speed, endurance, technique,
             blood_type, horoscope, molecular_density, existential_dread,
             fingers, toes, ancestors,
             wins, losses, draws,
             COALESCE(NULLIF(TRIM(lore),""),"") AS lore
      FROM fighters'
if [[ "$ONLY_ALIVE" == "1" ]]; then
  SQL+=" WHERE is_dead = 0"
fi
SQL+=';'

echo "[4/4] Creating missing fighter pages (no clobber)"
echo "[DEBUG] SQL Query: $SQL"
total_fighters=$(sqlite3 "$DB" "SELECT COUNT(*) FROM fighters;")
alive_fighters=$(sqlite3 "$DB" "SELECT COUNT(*) FROM fighters WHERE is_dead = 0;")
echo "[DEBUG] Total fighters in DB: $total_fighters (alive: $alive_fighters)"

# Store results in temp file to avoid subshell issues
tmpfile=$(mktemp)
sqlite3 -json "$DB" "$SQL" | jq -c '.[]' > "$tmpfile"
fighter_count=$(wc -l < "$tmpfile")
echo "[DEBUG] Processing $fighter_count fighters"

while read -r row; do
  id=$(jq -r '.id' <<<"$row")
  name=$(jq -r '.name' <<<"$row")
  team=$(jq -r '.team // ""' <<<"$row")
  class=$(jq -r '.class // ""' <<<"$row")
  strength=$(jq -r '.strength // 0' <<<"$row")
  speed=$(jq -r '.speed // 0' <<<"$row")
  endurance=$(jq -r '.endurance // 0' <<<"$row")
  technique=$(jq -r '.technique // 0' <<<"$row")
  blood_type=$(jq -r '.blood_type // ""' <<<"$row")
  horoscope=$(jq -r '.horoscope // ""' <<<"$row")
  molecular_density=$(jq -r 'if (.molecular_density == null) then "" else (.molecular_density|tostring) end' <<<"$row")
  existential_dread=$(jq -r '.existential_dread // ""' <<<"$row")
  fingers=$(jq -r '.fingers // ""' <<<"$row")
  toes=$(jq -r '.toes // ""' <<<"$row")
  ancestors=$(jq -r '.ancestors // ""' <<<"$row")
  wins=$(jq -r '.wins // 0' <<<"$row")
  losses=$(jq -r '.losses // 0' <<<"$row")
  draws=$(jq -r '.draws // 0' <<<"$row")
  lore=$(jq -r '.lore // ""' <<<"$row")

  DISPLAY_TITLE=$(printf "Roster #%03d %s" "$id" "$name")
  SAFE_NAME=$(printf '%s' "$name" | sed 's/[^A-Za-z0-9 _-]/_/g')
  URL_NAME=$(printf '%s' "$SAFE_NAME" | tr ' ' '_')
  TITLE=$(printf "Roster_%03d_%s" "$id" "$URL_NAME")

  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[DRY] Would create: $TITLE"
    continue
  fi

  [[ -z "$team" || "$team" == "null" ]] && team="Unaffiliated"
  [[ -z "$class" || "$class" == "null" ]] && class="Unclassified"
  [[ -z "$blood_type" || "$blood_type" == "null" ]] && blood_type="Unknown"
  [[ -z "$horoscope" || "$horoscope" == "null" ]] && horoscope="Classified"
  [[ -z "$molecular_density" || "$molecular_density" == "null" ]] && molecular_density="??"
  [[ -z "$existential_dread" || "$existential_dread" == "null" ]] && existential_dread="??"
  [[ -z "$fingers" || "$fingers" == "null" ]] && fingers="??"
  [[ -z "$toes" || "$toes" == "null" ]] && toes="??"
  [[ -z "$ancestors" || "$ancestors" == "null" ]] && ancestors="??"

  lore_safe=$(printf '%s' "$lore" | sed 's/|/{{!}}/g')
  if [[ -n "$PYTHON_BIN" ]]; then
    lore_markdown=$(printf '%s' "$lore_safe" | "$PYTHON_BIN" - "$DISPLAY_TITLE" <<'PY'
import sys,re
lore=sys.stdin.read()
title=sys.argv[1]
if not lore.strip():
    print("This fighter's lore file is missing. Please consult the Department of Narrative Risk.")
    sys.exit()

paragraphs=[p.strip() for p in re.split(r'\n\s*\n', lore) if p.strip()]
for p in paragraphs:
    if not p.endswith('.') and not p.endswith('!') and not p.endswith('?'):
        p += '.'
    print(p)
    print()
PY
    )
  else
    # simple shell fallback: split on blank lines
    lore_markdown=""
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ -z "$line" ]]; then
        lore_markdown+=$'\n'
      else
        lore_markdown+="$line"
        [[ "$line" =~ [.!?]$ ]] || lore_markdown+="."
        lore_markdown+=$'\n'
      fi
    done < <(printf '%s' "$lore_safe")
    lore_markdown+=$'\n'
  fi

  TEXT=$(cat <<EOF
{{DISPLAYTITLE:$DISPLAY_TITLE}}
{{Fighter
|team=$team
|class=$class
|strength=$strength
|speed=$speed
|endurance=$endurance
|technique=$technique
|lore=$lore
}}

'''$name''' is a registered combatant in the '''Spoodblort''' violence league. As '''$DISPLAY_TITLE''', this fighter represents '''$team''' and competes as a '''$class''' archetype.

== Lore ==
$lore_markdown

== Combat Profile ==
{| class="wikitable"
! Attribute !! Value
|-
| Strength || $strength
|-
| Speed || $speed
|-
| Endurance || $endurance
|-
| Technique || $technique
|}

== Chaos Metrics ==
{| class="wikitable"
! Metric !! Reading
|-
| Blood Type || $blood_type
|-
| Horoscope || $horoscope
|-
| Molecular Density || $molecular_density
|-
| Existential Dread || $existential_dread
|-
| Fingers || $fingers
|-
| Toes || $toes
|-
| Recorded Ancestors || $ancestors
|}

== Fight Record ==
* Wins: $wins
* Losses: $losses
* Draws: $draws

== Data Provenance ==
* Imported from Spoodblort game database on {{CURRENTDAY}} {{CURRENTMONTHNAME}} {{CURRENTYEAR}}.
* Lore managed by the Department of Narrative Risk.

[[Category:Fighters]]
EOF
)
  # Create or update the page every run (no skipping)
  retries=0
  while true; do
    resp=$(curl -s "$API?action=edit&format=json" -b "$COOKIE" \
      --data-urlencode "title=$TITLE" \
      --data-urlencode "text=$TEXT" \
      --data-urlencode "summary=Sync fighter page from game DB" \
      --data-urlencode "token=$CSRF")

    result=$(jq -r '.edit.result // empty' <<<"$resp")
    if [[ "$result" == "Success" ]]; then
      echo "✅ Synced: $TITLE"
      # Only try index update once to avoid rate limiting
      if [[ "$INDEX_UPDATED" != "1" ]]; then
        idx_status=0
        if ! add_to_fighters_index "$TITLE" "$DISPLAY_TITLE"; then
          idx_status=$?
        fi
        if [[ "$idx_status" == "2" ]]; then
          echo "[Index] Backing off due to rate limit"
          sleep 5
          RATE_MS=$((RATE_MS + 1000))
          INDEX_UPDATED=1  # Don't try again this run
        else
          INDEX_UPDATED=1  # Successfully updated, don't try again
        fi
      else
        echo "[Index] Skipping index update (already attempted this run)"
      fi
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

  if [[ "$RATE_MS" -gt 0 ]]; then
    sleep "$(awk -v ms="$RATE_MS" 'BEGIN{printf "%.3f", ms/1000}')"
  fi

done < "$tmpfile"

rm -f "$tmpfile"
echo "Done."