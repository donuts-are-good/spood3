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

# Options (hard-coded)
ONLY_ALIVE=0    # 1 to skip dead fighters
DRY_RUN=0       # 1 to preview titles only
RATE_MS=300     # throttle between edits (ms)

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
  local title="$1"  # e.g., Roster #062 Stone Cold Steve Austin

  # Fetch current page content (follow redirects; raw endpoint)
  local content
  content=$(curl -sL -H "User-Agent: SpoodblortBot/1.0" "$WIKI_BASE/wiki/Fighters?action=raw&ctype=text/plain")
  if [[ -z "$content" || "$content" == "null" ]]; then
    # Fallback to MediaWiki API
    content=$(curl -s "$API?action=query&prop=revisions&titles=Fighters&rvslots=main&rvprop=content&formatversion=2&format=json" -b "$COOKIE" | jq -r '.query.pages[0].revisions[0].slots.main.content // ""')
    if [[ -z "$content" ]]; then
      echo "[Index] Could not load Fighters page; skipping index update"
      return
    fi
  fi

  # If already present, skip
  if grep -Fq "[[$title]]" <(printf '%s' "$content"); then
    echo "[Index] Already listed: $title"
    return
  fi

  # Work with awk to locate the Notable Fighters list and keep bullets sorted
  local tmp
  tmp=$(mktemp)
  printf '%s' "$content" > "$tmp"

  local start end
  start=$(awk 'BEGIN{IGNORECASE=1} /^Notable[[:space:]]+Fighters:/{print NR; exit}' "$tmp") || true
  if [[ -z "$start" ]]; then
    echo "[Index] Could not find 'Notable Fighters:' heading; skipping index update"
    rm -f "$tmp"
    return
  fi

  # Find the end of the bullet block (blank line or non-bullet)
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

  bullets=$(printf '%s\n* [[%s]]\n' "$bullets" "$title" | awk 'NF')
  bullets=$(printf '%s\n' "$bullets" | sort -t# -k2,2n)

  newcontent=$(printf '%s\n%s\n%s' "$pre" "$bullets" "$others$post")

  resp=$(curl -s "$API?action=edit&format=json" -b "$COOKIE" \
    --data-urlencode "title=Fighters" \
    --data-urlencode "text=$newcontent" \
    --data-urlencode "summary=Add $title to Fighters index" \
    --data-urlencode "token=$CSRF")
  if [[ $(echo "$resp" | jq -r '.edit.result // empty') == "Success" ]]; then
    echo "[Index] Inserted into Fighters: $title"
  else
    echo "[Index] Edit failed for Fighters: $resp"
  fi

  rm -f "$tmp"
}

# Build SQL
SQL='SELECT id, name, team, fighter_class AS class, strength, speed, endurance, technique, COALESCE(NULLIF(TRIM(lore),""),"") AS lore FROM fighters'
if [[ "$ONLY_ALIVE" == "1" ]]; then
  SQL+=" WHERE is_dead = 0"
fi
SQL+=';'

echo "[4/4] Creating missing fighter pages (no clobber)"
sqlite3 -json "$DB" "$SQL" | jq -c '.[]' | while read -r row; do
  id=$(jq -r '.id' <<<"$row")
  name=$(jq -r '.name' <<<"$row")
  team=$(jq -r '.team // ""' <<<"$row")
  class=$(jq -r '.class // ""' <<<"$row")
  strength=$(jq -r '.strength // 0' <<<"$row")
  speed=$(jq -r '.speed // 0' <<<"$row")
  endurance=$(jq -r '.endurance // 0' <<<"$row")
  technique=$(jq -r '.technique // 0' <<<"$row")
  lore=$(jq -r '.lore // ""' <<<"$row")

  DISPLAY_TITLE=$(printf "Roster #%03d %s" "$id" "$name")
  SAFE_NAME=$(printf '%s' "$name" | sed 's/[^A-Za-z0-9 _-]/_/g')
  URL_NAME=$(printf '%s' "$SAFE_NAME" | tr ' ' '_')
  TITLE=$(printf "Roster_%03d_%s" "$id" "$URL_NAME")

  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[DRY] Would create: $TITLE"
    continue
  fi

  # Build page wikitext
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
[[Category:Fighters]]
EOF
)

  # Create or update the page every run (no skipping)
  resp=$(curl -s "$API?action=edit&format=json" -b "$COOKIE" \
    --data-urlencode "title=$TITLE" \
    --data-urlencode "text=$TEXT" \
    --data-urlencode "summary=Sync fighter page from game DB" \
    --data-urlencode "token=$CSRF")

  result=$(jq -r '.edit.result // empty' <<<"$resp")
  if [[ "$result" == "Success" ]]; then
    echo "✅ Synced: $TITLE"
    # Ensure it's on the Fighters index
    add_to_fighters_index "$TITLE"
  else
    echo "⚠️  Error for $TITLE: $resp"
  fi

  # throttle
  if [[ "$RATE_MS" -gt 0 ]]; then
    sleep "$(awk -v ms="$RATE_MS" 'BEGIN{printf "%.3f", ms/1000}')"
  fi

done

echo "Done."