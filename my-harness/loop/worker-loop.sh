#!/usr/bin/env bash
# loop/worker-loop.sh — L1 本地 worker loop（最小版本）
set -uo pipefail

ERROR_REPORT="/workshop/my-harness/test/image-parse-error.md"
TASK="修复 /workshop/my-harness/test/image-parse-error.md 中记录的错误。
错误报告内容如下：

$(cat "$ERROR_REPORT")

规则：只修改实现代码（Lambda 配置或代码），不要修改 test/ 下的任何文件。"

CONTEXT="$TASK"
MAX_ITER=5

echo "🔄 Worker Loop 启动 — 最多 $MAX_ITER 轮"
echo "停止条件：图片解析成功（DynamoDB status=done）"
echo ""

for i in $(seq 1 $MAX_ITER); do
  echo "━━━ 第 $i/$MAX_ITER 轮 ━━━"

  # 调用 Claude Code (headless/print 模式)
  claude -p "$CONTEXT" --dangerously-skip-permissions 2>/dev/null

  # 跑停止条件：重新上传图片触发 Lambda，检查 DynamoDB 结果
  echo "  🧪 运行 verify..."
  BUCKET="wrong-question-images-792539919723-prod"
  Q_ID="test-q-loop-$(date +%s)"
  KEY="images/test-user/${Q_ID}.jpeg"
  aws s3 cp '/workshop/my-harness/docs/original_1782354414262_303e7fcf3fa2a68efab8233f1706b6db.jpeg' \
    "s3://${BUCKET}/${KEY}" --region us-east-1 >/dev/null 2>&1
  sleep 15
  STATUS=$(aws dynamodb get-item \
    --table-name wrong-question-questions-prod \
    --key "{\"question_id\":{\"S\":\"${Q_ID}\"}}" \
    --region us-east-1 \
    --query "Item.status.S" --output text 2>&1)

  if [[ "$STATUS" == "done" ]]; then
    echo "  ✅ 验证通过！第 $i 轮收敛（status=done）"
    echo ""
    echo "=== LOOP 成功 === 退出码 0"
    exit 0
  else
    OUT="DynamoDB status=${STATUS}（期望 done）"
    echo "  ❌ 未通过 — 错误回灌下一轮（$OUT）"
    CONTEXT="$TASK

上一轮验证失败，错误如下，请据此修正（不要重复之前的尝试）：
$OUT"
  fi
  echo ""
done

echo "=== LOOP 未收敛 === 达到 $MAX_ITER 轮上限"
exit 1
