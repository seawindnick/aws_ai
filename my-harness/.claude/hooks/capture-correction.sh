#!/usr/bin/env bash
set -uo pipefail

REPO="/workshop/my-harness"
OUT="$REPO/memory/corrections.md"

INPUT=$(cat)

TEXT=$(echo "$INPUT" | jq -r '.prompt // ""' 2>/dev/null)
[ -z "$TEXT" ] && exit 0

if echo "$TEXT" | grep -qiE '不对|应该是|错了|别这么|搞反|不是这样|wrong|should be|don'\''t|stop|revert'; then
  DATETIME=$(date '+%Y-%m-%d %H:%M')
  EXCERPT=$(echo "$TEXT" | head -c 120)
  echo "- [$DATETIME] $EXCERPT" >> "$OUT"
fi

exit 0
