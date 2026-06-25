#!/usr/bin/env bash
set -uo pipefail

REPO="/workshop/my-harness"
MEMORY_DIR="$REPO/memory/daily"
mkdir -p "$MEMORY_DIR"

DATE=$(date '+%Y-%m-%d')
TIME=$(date '+%H:%M')
OUT="$MEMORY_DIR/$DATE.md"

FILES=$(git -C "$REPO" diff --name-only HEAD~1 2>/dev/null | tr '\n' ' ' | sed 's/ $//')
[ -z "$FILES" ] && FILES="(无改动)"

COMMITS=$(git -C "$REPO" log --oneline -3 --no-decorate 2>/dev/null | tr '\n' '|' | sed 's/|$//')
[ -z "$COMMITS" ] && COMMITS="(无 commit)"

{
  echo ""
  echo "## $TIME | session"
  echo "**改动文件：** $FILES"
  echo "**Git：** $COMMITS"
} >> "$OUT"
