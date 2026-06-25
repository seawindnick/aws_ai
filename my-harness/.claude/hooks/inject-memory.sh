#!/usr/bin/env bash
set -uo pipefail

REPO="/workshop/my-harness"

if [ -f "$REPO/memory/MEMORY.md" ]; then
  cat "$REPO/memory/MEMORY.md"
  echo ""
fi

if [ -f "$REPO/rules/personal.md" ]; then
  cat "$REPO/rules/personal.md"
  echo ""
fi

if [ -f "$REPO/memory/corrections.md" ] && [ -s "$REPO/memory/corrections.md" ]; then
  echo "## 近期纠正（待进化）"
  tail -10 "$REPO/memory/corrections.md"
fi
