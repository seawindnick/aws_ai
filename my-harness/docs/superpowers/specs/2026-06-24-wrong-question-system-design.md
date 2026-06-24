# 错题管理系统 设计文档

**日期**: 2026-06-24
**状态**: 已确认，待实现

---

## 一、项目概述

帮助学生拍照上传错题，AI 自动识别题目内容，数字化管理错题库，基于艾宾浩斯遗忘曲线智能推荐复习，并支持导出基本格式 PDF。

**用户角色**: 学生 / 教师 / 管理员（单校区，无多租户）
**认证方式**: Amazon Cognito 自建账号（邮箱+密码）
**部署平台**: AWS ECS Fargate
**后端语言**: Go 1.25
**前端框架**: React
**架构模式**: 前后端分离，Go 单体服务

---

## 二、整体架构

```
[前端 React · 静态文件]
            │  HTTPS
     [Go 1.25 单体服务 · ECS Fargate]
      ├── 认证模块（Cognito SDK）
      ├── 题目模块（上传→识别→入库）
      ├── 复习模块（遗忘曲线调度）
      ├── 推荐模块（Bedrock）
      └── 导出模块（PDF 生成）
            │
     ┌──────┼──────────┐
     │      │          │
  [RDS     [本地磁盘   [DynamoDB]
  MySQL]   /data/imgs] 复习调度记录]

     [第三方识别 API]   [Amazon Bedrock]
```

**关键决策：**
- Go 服务直接监听 HTTPS，无 API Gateway 中间层
- 图片保存到容器本地卷（ECS Task 挂载本地磁盘），路径存入数据库
- 认证由 Go 服务内 Cognito JWT 中间件统一处理
- 题目结构化数据存 RDS MySQL（utf8mb4，JSON 列存 topic_tags）
- 复习调度时间戳存 DynamoDB（高频读写 + TTL 自动清理）
- 数据按 `user_id` 隔离，无多租户

---

## 三、数据模型

### MySQL（结构化数据）

```sql
users
  id, cognito_sub, email, role(student/teacher/admin),
  school_name, created_at

questions
  id, user_id, image_path, raw_text, subject, topic_tags[],
  source(第三方API返回原始结果), created_at

errors
  id, user_id, question_id, wrong_count, last_wrong_at, created_at

review_records
  id, user_id, question_id, reviewed_at, result(pass/fail)
```

### DynamoDB（复习调度）

```
review_schedule
  PK: user_id
  SK: next_review_at（ISO 时间戳）
  question_id, interval_days（当前复习间隔天数）
  TTL: next_review_at + 30天
```

### 艾宾浩斯调度逻辑

- 初次错题 → `interval_days = 1`
- 复习通过 → `interval_days × 2`（1→2→4→8→16…）
- 复习失败 → `interval_days` 重置为 1
- 每次复习后写新 `review_schedule` 记录，旧记录 TTL 自动清除

### 图片存储

```
/data/imgs/{user_id}/{question_id}.jpg
```

路径存在 `questions.image_path`，Go 服务通过静态文件路由暴露访问。

---

## 四、API 路由设计

### 认证
```
POST /api/auth/register        注册（调 Cognito）
POST /api/auth/login           登录，返回 JWT
POST /api/auth/refresh         刷新 Token
```

### 题目管理
```
POST   /api/questions          上传图片 → 调第三方识别 → 入库
GET    /api/questions          列表（分页，支持按科目/标签筛选）
GET    /api/questions/:id      题目详情 + 图片
DELETE /api/questions/:id      删除题目及图片文件
```

### 错题记录
```
POST   /api/errors             记录一次答错（绑定 question_id）
GET    /api/errors             我的错题列表
```

### 复习
```
GET    /api/review/today       今日待复习题目（查 DynamoDB）
POST   /api/review/:id/result  提交复习结果（pass/fail），更新调度
```

### 推荐
```
GET    /api/recommend          调 Bedrock，返回推荐复习题目及理由
```

### 导出
```
POST   /api/export/pdf         按筛选条件生成 PDF，返回文件下载链接
```

### 中间件栈
```
Logger → Recovery → CORS → CognitoJWTAuth → RateLimiter → Handler
```
- `CognitoJWTAuth` 验证 JWT，解析 `user_id` 注入 context
- `/api/auth/*` 路由跳过 JWT 验证
- `Recovery` 只记录 panic 堆栈日志，不吞 error、不返回兜底 200

---

## 五、工程规则

### MUST 级别 — 不可逾越，违反即为 blocking bug

**R1 — 禁止吞 error**
所有 `error` 必须显式处理：要么返回给调用方，要么记录日志后终止当前流程。
禁止 `_ = someFunc()` 或 `if err != nil { return }` 不带任何日志/传播。

**R2 — 禁止自动兜底策略**
识别失败、Bedrock 调用失败、数据库写入失败一律返回明确错误给调用方，禁止静默降级（如识别失败自动填"未知题目"继续流程）。

**R3 — 第三方 API 响应必须校验再用**
识别 API 和 Bedrock 返回结果必须先做字段校验，缺失关键字段必须返回 error，禁止用零值/空字符串静默替代。（`confidence` 缺失例外：按遗忘曲线规则默认置 0）

**R4 — 错题数据必须按 userId 隔离**
所有查询必须在 service 层强制附加 `user_id = ctx.UserID` 条件，禁止仅凭客户端传入的 ID 查询，跨用户访问返回 `403`。

**R5 — HTTP 错误码必须语义正确**
- 未认证 → `401`，无权限 → `403`，资源不存在 → `404`，参数错误 → `400`，服务内部错误 → `500`
- 禁止所有错误统一返回 `200` + body 里藏 error 字段。

**R6 — 禁止在 Handler 层直接操作数据库**
严格遵守 Handler → Service → Repository 三层，禁止 Handler 直接调用 DB/ORM。

**R7 — 图片路径禁止拼接用户输入**
图片存储路径必须由服务端用 `question_id`（UUID）生成，禁止将用户上传的文件名或任何用户输入拼入文件路径，防止路径穿越攻击。

**R8 — AI 模型必须使用 claude-sonnet-4-6 `[MUST]`**
本项目所有 Bedrock / Claude API 调用必须指定模型 `claude-sonnet-4-6`，禁止使用其他模型 ID 或依赖默认模型。Claude Code 开发环境同样通过 `.claude/settings.json` 锁定为此模型。

---

### SHOULD 级别 — 强烈建议，偏离需说明理由

**S1 — error 应携带上下文**
使用 `fmt.Errorf("查询题目失败: %w", err)` 包装错误，保留调用链，便于排查。

**S2 — 数据库写操作应使用事务**
涉及多表写入应在同一事务内完成，单表写操作可例外。

**S3 — 所有外部调用应设置超时**
第三方识别 API、Bedrock、DynamoDB 调用必须通过带 `context.WithTimeout` 的 ctx 传入，超时值在配置文件中定义，禁止硬编码。

**S4 — Handler 层应做输入校验**
请求参数在 Handler 层校验（类型、必填、长度），校验不通过直接返回 `400`，不应将非法参数透传到 Service 层。

**S5 — 日志应结构化输出**
使用结构化日志（如 `slog`），每条日志带 `user_id`、`request_id`、`error` 字段，禁止裸 `fmt.Println` 打日志。

**S6 — 敏感配置不应硬编码**
Cognito Pool ID、第三方 API Key、DB 连接串等必须从环境变量或 AWS Secrets Manager 读取，禁止提交到代码仓库。

---

### MAY 级别 — 可选指导，视情况采用

**M1 — 可为高频查询添加缓存**
今日复习列表、推荐结果可加内存缓存，缓存失效必须有明确策略，不可无限期缓存。

**M2 — 可对第三方 API 添加重试**
识别 API 网络抖动可重试，重试次数和间隔必须在配置中显式声明，禁止无限重试，重试耗尽后必须返回 error（不得兜底，见 R2）。

**M3 — 可为题目打自定义标签**
除第三方 API 返回的 `topic_tags` 外，用户可手动追加自定义标签，供筛选和导出使用。

**M4 — 可记录 API 响应时间指标**
可在中间件层记录每个接口耗时，输出到 CloudWatch Metrics，用于性能监控。
