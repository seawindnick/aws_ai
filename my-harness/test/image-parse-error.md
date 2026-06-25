# 图片解析执行报告 — 2026-06-25

## 测试图片
`docs/original_1782354414262_303e7fcf3fa2a68efab8233f1706b6db.jpeg` (61.1 KiB)

## 执行步骤
1. 上传图片到 S3：`s3://wrong-question-images-792539919723-prod/images/test-user/test-q-parse-001.jpeg`
2. S3 PUT 事件触发 Lambda `wrong-question-image-analyzer-prod`
3. Lambda 调用 Bedrock 进行图片分析
4. 查询 DynamoDB `wrong-question-questions-prod` 获取结果

## 结果

**DynamoDB 状态：** `failed`

```json
{
  "question_id": "test-q-parse-001",
  "status": "failed",
  "updated_at": "2026-06-25T08:00:34Z"
}
```

## 错误信息

**Lambda 日志（RequestId: 238639c9-3f47-4f5d-8485-9a5f92fa45a7）：**

```
process image failed
key: images/test-user/test-q-parse-001.jpeg
error: bedrock invoke: operation error Bedrock Runtime: InvokeModel,
       https response error StatusCode: 404, RequestID: e69f51d4-f189-475d-9a9c-5af74020b635,
       ResourceNotFoundException: This model version has reached the end of its life.
       Please refer to the AWS documentation for more details.
```

## 根因分析

`image-analyzer` Lambda 配置的视觉模型 ID 已到期下线：

| 项 | 值 |
|---|---|
| 当前配置（`VISION_MODEL_ID`） | `anthropic.claude-3-5-sonnet-20241022-v2:0` |
| AWS 返回状态码 | 404 ResourceNotFoundException |
| 错误原因 | 该模型版本已 End-of-Life，Bedrock 不再提供服务 |

Lambda 代码路径：`lambda/image-analyzer/main.go:45`
```go
visionModelID = getEnv("VISION_MODEL_ID", "anthropic.claude-3-5-sonnet-20241022-v2:0")
```

## 未修复（按要求仅记录）
需将 `VISION_MODEL_ID` 更新为当前可用的 Bedrock 模型 ID。
