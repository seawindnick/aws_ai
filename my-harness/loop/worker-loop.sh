#!/usr/bin/env bash
# loop/worker-loop.sh — L1 本地 worker loop（最小版本）
set -uo pipefail

TASK="修复 src/ 中识别功能的 bug，使 ci/verify.sh 的契约断言全部通过。
规则：不要修改测试代码、不要修改断言、不要修改 ci/verify.sh。只改实现。"

CONTEXT="$TASK"
MAX_ITER=5

echo "🔄 Worker Loop 启动 — 最多 $MAX_ITER 轮"
echo "停止条件：ci/verify.sh 通过（退出码 0）"
echo ""

for i in $(seq 1 $MAX_ITER); do
  echo "━━━ 第 $i/$MAX_ITER 轮 ━━━"

  # 调用 Claude Code (headless/print 模式)
  claude -p "$CONTEXT" --dangerously-skip-permissions 2>/dev/null

  # 跑停止条件
  echo "  🧪 运行 verify..."
  if OUT=$(./ci/verify.sh 2>&1); then
    echo "  ✅ 验证通过！第 $i 轮收敛"
    echo ""
    echo "=== LOOP 成功 === 退出码 0"
    exit 0
  else
    echo "  ❌ 未通过 — 错误回灌下一轮"
    CONTEXT="$TASK

上一轮验证失败，错误如下，请据此修正（不要重复之前的尝试）：
$OUT"
  fi
  echo ""
done

echo "=== LOOP 未收敛 === 达到 $MAX_ITER 轮上限"
exit 1
