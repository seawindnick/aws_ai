# 错题本 — 系统设计文档

**版本**: 1.2
**日期**: 2026-06-24
**基于需求**: requirements.md v1.1
**技术栈**: Go 1.25 · GORM · MySQL · Vue 3 · AWS ECS Fargate

---

## 一、整体架构

```
[Vue 3 前端 · 静态文件托管]
          │  HTTPS / JSON API
  [Go 1.25 单体服务 · ECS Fargate · :8080]
          │
    ┌─────┼──────────────┬────────────────┐
    │     │              │                │
 [MySQL  [本地磁盘       [DynamoDB        [S3 Vectors]
  RDS]   /data/imgs]    review_schedules  向量存储 + kNN 查询
                         embedding_jobs]

  [第三方识别 API]   [Amazon Bedrock claude-sonnet-4-6]
                    [Amazon Bedrock Titan Embeddings v2]

  [Lambda Worker] ←─ DynamoDB Stream (embedding_jobs)
       └─→ Bedrock Titan → S3 Vectors
```

**关键决策：**
- Go 单体服务，Handler → Service → Repository 三层，无多余抽象
- GORM 作为 MySQL ORM，减少手写 SQL
- DynamoDB 用于复习调度（高频读写 + TTL）和 embedding 异步作业队列，其余全部在 MySQL
- 语义搜索采用异步 Lambda Worker：题目上传后写 embedding_jobs 至 DynamoDB，DynamoDB Stream 触发 Lambda 生成向量并写入 S3 Vectors；查询时由主服务直接调用 S3 Vectors QueryVectors 接口（Serverless，按查询计费，无常驻集群）
- 图片存储到 ECS Task 挂载的本地卷，路径写入 MySQL（此为简化设计；生产环境应迁移至 S3 以避免 Task 重启或多实例扩容时数据丢失）
- 前端 Vue 3 + Vite，调用后端 JSON API，独立部署

---

## 二、目录结构

```
backend/
  main.go                  启动入口
  config/config.go         配置读取（环境变量）
  db/
    mysql.go               GORM 初始化
    dynamodb.go            DynamoDB 客户端
  model/                   GORM 模型（数据库表定义）
    user.go
    question.go
    tag.go
    paper.go
    notification.go
    error_record.go
    review.go
    class.go
    task.go
  handler/                 HTTP 处理器（路由入口）
    auth.go
    question.go
    tag.go
    paper.go
    review.go
    recommend.go
    notification.go
    stats.go
    admin.go
    me.go
    class.go
    task.go
  service/                 业务逻辑
    auth.go
    question.go
    tag.go
    paper.go
    review.go
    recommend.go
    notification.go
    stats.go
    admin.go
    class.go
    task.go
  middleware/
    auth.go                JWT 校验，注入 userID
    role.go                角色权限检查
  migrations/              SQL 迁移文件

frontend/
  src/
    views/                 页面组件
    components/            公共组件
    api/                   axios 封装
    router/                vue-router
    store/                 pinia 状态管理
```

---

## 三、数据模型

所有表使用 GORM AutoMigrate 或手动迁移文件管理。

### users
```go
type User struct {
    ID           string    `gorm:"primaryKey;type:varchar(36)"`
    CognitoSub   string    `gorm:"uniqueIndex;type:varchar(128)"`
    Email        string    `gorm:"uniqueIndex;type:varchar(255)"`
    Nickname     string    `gorm:"type:varchar(100)"`
    Role         string    `gorm:"type:enum('student','teacher','admin');default:'student'"`
    Status       string    `gorm:"type:enum('active','inactive');default:'active'"`
    DeactivatedAt *time.Time
    CreatedAt    time.Time
}
```

### questions
```go
type Question struct {
    ID           string    `gorm:"primaryKey;type:varchar(36)"`
    UserID       string    `gorm:"index;type:varchar(36)"`
    ImagePath    string    `gorm:"type:varchar(500)"`
    RawText      string    `gorm:"type:text"`
    Subject      string    `gorm:"type:varchar(100)"`
    Category     string    `gorm:"type:enum('multiple_choice','fill_blank','essay','true_false','calculation','unknown');default:'unknown'"`
    Status       string    `gorm:"type:enum('pending_review','approved','rejected');default:'pending_review'"`
    Confidence   float64
    ReviewNote   string    `gorm:"type:text"`
    ReviewedBy   string    `gorm:"type:varchar(36)"`
    ReviewedAt   *time.Time
    CreatedAt    time.Time
}
```

### question_tags
```go
type QuestionTag struct {
    ID         string    `gorm:"primaryKey;type:varchar(36)"`
    QuestionID string    `gorm:"index;type:varchar(36)"`
    UserID     string    `gorm:"type:varchar(36)"`
    Name       string    `gorm:"type:varchar(100)"`
    Status     string    `gorm:"type:enum('suggested','confirmed');default:'suggested'"`
    CreatedAt  time.Time
}
// 唯一约束：(question_id, name)
```

### papers
```go
type Paper struct {
    ID        string    `gorm:"primaryKey;type:varchar(36)"`
    UserID    string    `gorm:"index;type:varchar(36)"`
    Title     string    `gorm:"type:varchar(200)"`
    Status    string    `gorm:"type:enum('draft','published');default:'draft'"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### paper_questions
```go
type PaperQuestion struct {
    ID         string    `gorm:"primaryKey;type:varchar(36)"`
    PaperID    string    `gorm:"index;type:varchar(36)"`
    QuestionID string    `gorm:"type:varchar(36)"`
    Position   int
    CreatedAt  time.Time
}
// 唯一约束：(paper_id, question_id)
```

### error_records
```go
type ErrorRecord struct {
    ID          string    `gorm:"primaryKey;type:varchar(36)"`
    UserID      string    `gorm:"index;type:varchar(36)"`
    QuestionID  string    `gorm:"type:varchar(36)"`
    WrongCount  int       `gorm:"default:1"`
    LastWrongAt time.Time
    CreatedAt   time.Time
}
// 唯一约束：(user_id, question_id)
```

### review_records
```go
type ReviewRecord struct {
    ID         string    `gorm:"primaryKey;type:varchar(36)"`
    UserID     string    `gorm:"index;type:varchar(36)"`
    QuestionID string    `gorm:"type:varchar(36)"`
    Result     string    `gorm:"type:enum('pass','fail')"`
    ReviewedAt time.Time
}
```

### notifications
```go
type Notification struct {
    ID        string    `gorm:"primaryKey;type:varchar(36)"`
    UserID    string    `gorm:"index;type:varchar(36)"`
    Type      string    `gorm:"type:varchar(50)"`
    Title     string    `gorm:"type:varchar(200)"`
    Body      string    `gorm:"type:text"`
    RefID     string    `gorm:"type:varchar(36)"`
    IsRead    bool      `gorm:"default:false"`
    CreatedAt time.Time
}
```

### classes
```go
type Class struct {
    ID         string    `gorm:"primaryKey;type:varchar(36)"`
    Name       string    `gorm:"type:varchar(100)"`
    TeacherID  string    `gorm:"index;type:varchar(36)"`
    InviteCode string    `gorm:"uniqueIndex;type:varchar(6)"`
    CreatedAt  time.Time
}
```

### class_members
```go
type ClassMember struct {
    ID       string    `gorm:"primaryKey;type:varchar(36)"`
    ClassID  string    `gorm:"index;type:varchar(36)"`
    UserID   string    `gorm:"type:varchar(36)"`
    JoinedAt time.Time
}
// 唯一约束：(class_id, user_id)
```

### class_tasks
```go
type ClassTask struct {
    ID         string     `gorm:"primaryKey;type:varchar(36)"`
    ClassID    string     `gorm:"index;type:varchar(36)"`
    PaperID    string     `gorm:"type:varchar(36)"`
    Title      string     `gorm:"type:varchar(200)"`
    AssignedBy string     `gorm:"type:varchar(36)"`
    DueAt      *time.Time
    Status     string     `gorm:"type:enum('active','closed');default:'active'"`
    CreatedAt  time.Time
}
```

### task_submissions
```go
type TaskSubmission struct {
    ID          string    `gorm:"primaryKey;type:varchar(36)"`
    TaskID      string    `gorm:"index;type:varchar(36)"`
    UserID      string    `gorm:"type:varchar(36)"`
    QuestionID  string    `gorm:"type:varchar(36)"`
    Result      string    `gorm:"type:enum('pass','fail')"`
    SubmittedAt time.Time
}
// 唯一约束：(task_id, user_id, question_id)
```

### DynamoDB — review_schedules
```
PK: user_id (String)
SK: question_id (String)
next_review_at: ISO8601 字符串
interval_days:  Number
TTL:            Unix 时间戳（next_review_at + 30天）

GSI: user_date_index
  PK: user_id (String)
  SK: next_review_at (String)  ← ISO8601 字典序 = 时间序，支持 ≤ today 范围查询
```

> 查询今日待复习使用 GSI `user_date_index`：`PK=user_id AND SK ≤ today`，避免全分区扫描。

---

## 四、API 路由

> 规则：只使用 GET / POST 两种方法。资源 ID 通过请求体（POST）或 Query 参数（GET）传递，不放在路径中。

### 认证（公开）
```
POST /api/auth/login             body: {email, password}
POST /api/auth/refresh           body: {refresh_token}
```

### 用户自管理（需登录）
```
GET  /api/me                     返回当前用户信息
POST /api/me/nickname            body: {nickname}
POST /api/me/password            body: {old_password, new_password}
POST /api/me/deactivate          body: {password}  注销账号
```

### 题目（需登录）
```
POST /api/questions/upload       body: multipart  上传单张图片
POST /api/questions/batch        body: multipart  批量上传
GET  /api/questions/list         query: subject, status, page, page_size
GET  /api/questions/search       query: subject, tag, category, keyword, date_from, date_to, status, page, page_size
GET  /api/questions/detail       query: id
POST /api/questions/delete       body: {id}
POST /api/questions/category     body: {id, category}
```

### 标签（需登录）
```
GET  /api/tags/list              query: question_id
POST /api/tags/add               body: {question_id, name}
POST /api/tags/confirm           body: {tag_id}
POST /api/tags/delete            body: {tag_id}
```

### 试卷（需登录）
```
POST /api/papers/create          body: {title}
GET  /api/papers/list            query: page, page_size
GET  /api/papers/detail          query: id
POST /api/papers/rename          body: {id, title}
POST /api/papers/delete          body: {id}
POST /api/papers/add-question    body: {paper_id, question_id, position}
POST /api/papers/remove-question body: {paper_id, question_id}
POST /api/papers/reorder         body: {paper_id, positions:[{question_id, position}]}
GET  /api/papers/questions       query: paper_id
POST /api/papers/duplicate       body: {paper_id}
POST /api/papers/export          body: {paper_id}
```

### 复习调度（需登录）
```
GET  /api/review/today           今日待复习题目列表
POST /api/review/submit          body: {question_id, result}  pass/fail
```

### 错题记录（需登录）
```
GET  /api/error-records/list     query: page, page_size
```

### 通知（需登录）
```
GET  /api/notifications/list     query: unread, page, page_size
POST /api/notifications/read     body: {id}
POST /api/notifications/read-all 无 body
```

### 统计（需登录）
```
GET  /api/stats/me               query: date_from, date_to
GET  /api/stats/class            query: date_from, date_to          （teacher/admin）
GET  /api/stats/student          query: student_id, date_from, date_to  （teacher/admin）
```

### AI 推荐（需登录）
```
GET  /api/recommend              返回最多 5 条推荐题目
```

### 人工审核（teacher/admin）
```
GET  /api/review-queue/list      query: page, page_size
POST /api/review-queue/submit    body: {question_id, action, category, note}
```

### 管理员（admin）
```
POST /api/admin/users/create     body: {email, role}
POST /api/admin/users/import     body: multipart CSV
GET  /api/admin/users/list       query: role, status, page, page_size
POST /api/admin/users/status     body: {user_id, status}
POST /api/admin/users/role       body: {user_id, role}
GET  /api/admin/users/questions  query: user_id, page, page_size
```

### 班级（需登录）
```
POST /api/classes/create         body: {name}                        （teacher）
GET  /api/classes/list           我的班级列表
GET  /api/classes/detail         query: class_id
POST /api/classes/remove-member  body: {class_id, user_id}           （teacher）
POST /api/classes/reset-code     body: {class_id}                    （teacher）
POST /api/classes/join           body: {invite_code}                  （student）
POST /api/classes/leave          body: {class_id}                    （student）
```

### 任务（需登录）
```
POST /api/tasks/create           body: {class_id, paper_id, title, due_at}  （teacher）
GET  /api/tasks/list             query: class_id
POST /api/tasks/update           body: {task_id, due_at, status}             （teacher）
GET  /api/tasks/progress         query: task_id                              （teacher）
GET  /api/tasks/detail           query: task_id
POST /api/tasks/submit           body: {task_id, results:[{question_id, result}]}  （student）
```

---

## 五、分层说明

### Handler 层职责
- 解析请求参数，校验格式（必填、类型、长度）
- 从 JWT context 取 userID，不信任客户端传入的 user_id
- 调用 Service，将结果序列化为 JSON 返回
- 不直接操作数据库

### Service 层职责
- 业务规则（置信度分级、Ebbinghaus 计算、权限二次校验）
- 编排多步操作（上传→识别→入库→创建标签）
- 调用外部 API（识别、Bedrock、Cognito）

### Repository 层（GORM）
- 封装所有数据库操作，位于 `backend/internal/repository/`
- 所有查询强制带 `user_id` 条件（REQ-SEC-01）
- 不含业务逻辑

---

## 六、认证方案

使用 Amazon Cognito User Pool：
- 账号由 admin 通过 `POST /api/admin/users/create` 创建（REQ-ACCT-01），不支持用户自助注册
- admin 创建时后端调 Cognito AdminCreateUser，生成随机初始密码，设置 `FORCE_CHANGE_PASSWORD` 状态，初始密码在创建响应中返回一次
- 用户首次登录时 Cognito 返回 `NEW_PASSWORD_REQUIRED` Challenge；前端跳转 `FirstLogin.vue` 引导修改密码，完成后正常颁发 JWT
- 后续登录调 Cognito SDK，返回 JWT（AccessToken + RefreshToken）
- 后端 `middleware/auth.go` 用 JWKS 校验 JWT 签名，解析 `sub` 作为 userID 注入 context
- 角色存在本地 `users.role`，不依赖 Cognito 自定义属性

```go
// middleware 伪代码
func JWTAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization") // Bearer xxx
        claims, err := verifyJWT(token)         // JWKS 验签
        if err != nil { writeError(w, 401); return }
        ctx := context.WithValue(r.Context(), "userID", claims.Sub)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 七、图片上传流程

```
1. Handler 校验文件格式（JPEG/PNG magic bytes）和大小（1KB~10MB）
2. Service 生成 UUID 作为文件名，写入 /data/imgs/{userID}/{uuid}.jpg
3. 调用第三方识别 API，获取 raw_text / subject / topic_tags / confidence
4. 按 confidence 分级：
   ≥ 0.85 → approved
   0.50~0.85 → pending_review
   < 0.50 → 删除文件，返回 422
5. GORM 写入 questions 表
6. 识别返回的 topic_tags 批量写入 question_tags（status=suggested）
7. 写入 DynamoDB embedding_jobs{question_id, user_id, status=pending}
   （异步触发 Lambda Worker 生成向量；写入失败记录日志，不阻断主流程）
```

---

## 八、Ebbinghaus 复习调度

```
提交复习结果（pass/fail）：
  1. GET DynamoDB(PK=user_id, SK=question_id)
     - 有记录 → 取 interval_days
     - 无记录（首次提交）→ interval_days = 0（初始基准）

  2. 计算新 interval：
     pass → new_interval = max(interval_days, 1) × 2
            首次 pass：0 → 1 → next=1天后
     fail → new_interval = 1

  3. next_review_at = now() + new_interval 天
     写入 DynamoDB（PK=user_id, SK=question_id），覆盖旧记录
     TTL = next_review_at + 30天

查询今日待复习：
  DynamoDB Query GSI user_date_index:
    PK=user_id, SK ≤ today（ISO8601 字典序范围查询）
```

---

## 九、科目掌握率计算

```
对每道 approved 题目取 DynamoDB 中的 interval_days：
  weight = log2(interval_days + 1)
  max_weight = log2(65)  // 64 天上限：pass 连续翻倍 6 次后（1→2→4→8→16→32→64）认为已完全掌握

mastery_rate(科目) = Σweight / (题目数 × max_weight)
结果截断到 [0.0, 1.0]，保留两位小数
```

> 无复习历史的题目 `interval_days` 视为 0，`weight = log2(1) = 0`，符合 REQ-STATS-02。

---

## 十、错误响应规范

所有错误统一格式：
```json
{"error": "human-readable message"}
```

| 场景 | 状态码 |
|------|--------|
| 参数错误 | 400 |
| 未认证 | 401 |
| 无权限 | 403 |
| 资源不存在 | 404 |
| 状态冲突（重复提交等） | 409 |
| 识别置信度过低 | 422 |
| 外部服务失败 | 502 |
| 其他内部错误 | 500 |

---

## 十一、配置（环境变量）

```
PORT                    服务端口，默认 8080
DB_DSN                  MySQL 连接串
AWS_REGION              AWS 区域
COGNITO_USER_POOL_ID    Cognito User Pool ID
COGNITO_CLIENT_ID       Cognito App Client ID
DYNAMO_TABLE_SCHEDULE   DynamoDB 表名
RECOGNITION_API_URL     第三方识别 API 地址
RECOGNITION_API_KEY     识别 API 密钥
BEDROCK_MODEL_ID        claude-sonnet-4-6
IMAGE_DIR               图片存储根目录，默认 /data/imgs
EXPORT_DIR              PDF 导出目录，默认 /data/exports
EXTERNAL_TIMEOUT_SEC    外部 API 超时秒数，默认 30
SECRETS_MANAGER_ARN     AWS Secrets Manager ARN（可选；设置后 DB_DSN/API Key 从 Secrets Manager 读取，优先级高于同名环境变量）
```

> **安全约束（REQ-SEC-03）**：所有凭据（数据库连接串、识别 API Key 等）须通过环境变量或 AWS Secrets Manager 注入，不得硬编码在源代码中。

---

## 十二、前端结构（Vue 3）

```
src/
  api/
    auth.js         登录/刷新（无自助注册；账号由 admin 创建）
    question.js     题目 CRUD、上传、搜索
    paper.js        试卷编排、导出
    review.js       复习调度
    stats.js        统计看板
    class.js        班级、任务
    admin.js        管理员接口
  views/
    Login.vue
    FirstLogin.vue        首次登录改密（admin 创建账号后，用户首次登录强制修改初始密码）
    QuestionList.vue      题库列表 + 搜索
    QuestionDetail.vue    题目详情 + 标签管理
    Upload.vue            拍照上传
    PaperList.vue         试卷列表
    PaperEditor.vue       试卷编排
    ReviewToday.vue       今日复习
    Stats.vue             学习统计看板
    ClassList.vue         班级列表
    TaskDetail.vue        任务详情 + 提交
    AdminUsers.vue        管理员用户管理
    ReviewQueue.vue       人工审核队列（teacher/admin）
  store/
    user.js       当前登录用户信息、token
  router/
    index.js      路由定义 + 登录守卫
```

JWT token 存储在 `localStorage`，每次请求由 axios 拦截器自动加 `Authorization: Bearer` 头。Token 过期时拦截器自动调 refresh 接口续期。

---

## 十三、依赖清单

### 后端
```
go 1.25
gorm.io/gorm
gorm.io/driver/mysql
github.com/go-chi/chi/v5
github.com/go-chi/cors
github.com/google/uuid
github.com/lestrrat-go/jwx/v2       JWT 验签
github.com/aws/aws-sdk-go-v2        Cognito / Bedrock / DynamoDB / S3 Vectors
github.com/jung-kurt/gofpdf         PDF 生成
```

### 前端
```
vue 3
vite
vue-router 4
pinia
axios
```

---

## 十四、模块交互图（请求处理层次）

```
┌──────────────────────────────────────────────────────────────┐
│                       前端 Vue 3                              │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTPS / JSON API
┌──────────────────────────────▼───────────────────────────────┐
│                   Go 单体服务 (:8080)                         │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │           Middleware  JWT校验 · 角色检查                  │ │
│  └──────────────────────────┬────────────────────────────── ┘ │
│                             │                                 │
│  ┌──────────────────────────▼──────────────────────────────┐ │
│  │                     Handler 层                           │ │
│  │  auth  question  tag   paper   review   recommend        │ │
│  │  notification  stats  class   task   admin   me          │ │
│  └──────────────────────────┬────────────────────────────── ┘ │
│                             │                                 │
│  ┌──────────────────────────▼──────────────────────────────┐ │
│  │                     Service 层                           │ │
│  │  auth  question  tag   paper   review   recommend        │ │
│  │  notification  stats  class   task   admin               │ │
│  │                                                          │ │
│  │  跨 Service 调用:                                        │ │
│  │    question ──► tag       上传后创建 suggested 标签      │ │
│  │    review   ──► notif     审核完成发站内通知             │ │
│  │    task     ──► review    任务提交触发复习调度           │ │
│  └──────────┬───────────────────────────┬───────────────────┘ │
└─────────────┼───────────────────────────┼─────────────────────┘
              │                           │
              ▼                           ▼
┌─────────────────────────┐   ┌──────────────────────────────────┐
│        存储层            │   │           外部服务                │
│                         │   │                                  │
│  MySQL (RDS)            │   │  Amazon Cognito  认证 / 账号管理  │
│  所有业务结构化数据       │   │  第三方识别 API  图片 → 题目文本  │
│                         │   │  Amazon Bedrock  AI 推荐         │
│  DynamoDB               │   │    claude-sonnet-4-6             │
│  复习调度 + TTL 自动清理  │   └──────────────────────────────────┘
│                         │
│  本地磁盘 /data/imgs     │
│  原始图片文件            │
└─────────────────────────┘
```

---

## 十五、数据库表结构与关系

### 表结构

```
  users                              questions
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  UK  cognito_sub     varchar(128)   FK  user_id          varchar(36)
  UK  email           varchar(255)       image_path       varchar(500)
      nickname        varchar(100)       raw_text         text
      role            enum               subject          varchar(100)
        student / teacher / admin        category         enum
      status          enum                 multiple_choice / fill_blank
        active / inactive                  essay / true_false
      deactivated_at  datetime             calculation / unknown
      created_at      datetime           status           enum
                                           pending_review / approved
                                           rejected
                                         confidence       decimal
                                    FK  reviewed_by      varchar(36)
                                         reviewed_at     datetime
                                         created_at      datetime

  question_tags                      papers
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  FK  question_id     varchar(36)    FK  user_id          varchar(36)
  FK  user_id         varchar(36)        title            varchar(200)
      name            varchar(100)       status           enum
      status          enum                 draft / published
        suggested / confirmed            created_at      datetime
      created_at      datetime           updated_at      datetime
  UK  (question_id, name)

  paper_questions                    error_records
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  FK  paper_id        varchar(36)    FK  user_id          varchar(36)
  FK  question_id     varchar(36)    FK  question_id      varchar(36)
      position        int                wrong_count      int
      created_at      datetime           last_wrong_at   datetime
  UK  (paper_id, question_id)            created_at      datetime
                                    UK  (user_id, question_id)

  review_records                     notifications
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  FK  user_id         varchar(36)    FK  user_id          varchar(36)
  FK  question_id     varchar(36)        type            varchar(50)
      result          enum               title           varchar(200)
        pass / fail                      body            text
      reviewed_at     datetime           ref_id          varchar(36)
                                         is_read         bool
                                         created_at      datetime

  classes                            class_members
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  FK  teacher_id      varchar(36)    FK  class_id         varchar(36)
      name            varchar(100)   FK  user_id          varchar(36)
  UK  invite_code     varchar(6)         joined_at       datetime
      created_at      datetime       UK  (class_id, user_id)

  class_tasks                        task_submissions
  ─────────────────────────────────  ──────────────────────────────────────
  PK  id              varchar(36)    PK  id              varchar(36)
  FK  class_id        varchar(36)    FK  task_id          varchar(36)
  FK  paper_id        varchar(36)    FK  user_id          varchar(36)
  FK  assigned_by     varchar(36)    FK  question_id      varchar(36)
      title           varchar(200)       result           enum
      due_at          datetime             pass / fail
      status          enum               submitted_at    datetime
        active / closed             UK  (task_id, user_id, question_id)
      created_at      datetime

  DynamoDB: review_schedules
  ──────────────────────────────────────────────────────────────────
  PK  user_id         String
  SK  question_id     String
      next_review_at  String  ISO8601
      interval_days   Number
      TTL             Number  Unix 时间戳 (next_review_at + 30天)
```

### 外键关系一览

```
  主表              从表                 关系说明
  ──────────────    ──────────────────   ──────────────────────────
  users        1──* questions            用户拥有多道题目
  users        1──* question_tags        用户拥有多个标签
  users        1──* papers               用户拥有多张试卷
  users        1──* error_records        用户产生多条错题记录
  users        1──* review_records       用户提交多条复习记录
  users        1──* notifications        用户接收多条通知
  users        1──* class_members        用户加入多个班级
  users        1──* task_submissions     用户提交多条任务结果
  users        1──* classes              教师教授多个班级

  questions    1──* question_tags        题目拥有多个标签
  questions    1──* paper_questions      题目出现在多张试卷
  questions    1──* error_records        题目关联多条错题记录
  questions    1──* review_records       题目关联多条复习记录
  questions    1──* task_submissions     题目关联多条任务提交

  papers       1──* paper_questions      试卷包含多道题目
  papers       1──* class_tasks          试卷用于多个班级任务

  classes      1──* class_members        班级包含多个成员
  classes      1──* class_tasks          班级发布多个任务

  class_tasks  1──* task_submissions     任务收到多条提交
```

---

## 十六、功能模块关系图

```
  说明:  ──►  强依赖（核心流程）     - -►  触发 / 通知（弱依赖）

  ┌──────────────────────────────────────────────────────────────┐
  │  认证模块 (Cognito JWT)                                       │
  │  提供 userID，所有请求必须先经过认证中间件                     │
  └──────────────────────────────────────────────────────────────┘
        │                              │
        ▼                              ▼
  ┌───────────────┐            ┌───────────────────┐
  │   账号管理     │            │    用户自管理       │
  │ 管理员         │            │ 改密码/昵称/注销   │
  │ 创建/停用/     │            └───────────────────┘
  │ 改角色         │
  └───────────────┘

  ─────────────────── 题目核心流程 ──────────────────────────────

  ┌──────────┐         ┌──────────┐
  │ 上传识别  │ ──────► │ 标签管理  │
  │ 图片→题目 │         │suggest/  │
  └────┬──┬──┘         │ confirm  │
       │  │            └────┬─────┘
       │  │ approved/       │ 标签可用于过滤
       │  │ pending_review  │
       │  │                 ▼
       │  │           ┌───────────────────────────────────────┐
       │  └─────────► │              搜索                      │
       │              │  关键词搜索  多维过滤（subject/tag/     │
       │              │             category/date/status）     │
       │ 低置信度      │                                        │
       │              │  语义搜索（独立接口）                   │
       │              │  query → Bedrock Titan Embed           │
       │              │       → OpenSearch kNN(user_id filter) │
       │              │       → Top-5 + score                  │
       │              └────────────────┬──────────────────────┘
       │                               │ 搜索条件复用
       ▼                               ▼
  ┌──────────┐ - - ► ┌──────────┐  ┌──────────────┐
  │ 人工审核  │       │   通知   │  │   试卷编排   │
  │ 审核队列  │ 审核  │ 站内消息  │  │  草稿/发布   │
  └──────────┘ 完成  └──────────┘  └──────┬───────┘
                                           │          │
                                    ──────► ▼          ▼
                                    仅含  ┌──────────┐
                                    已审  │ PDF 导出  │
                                    核题  └──────────┘
                                           │
                                           │ 发布为班级任务

  ─────────────────── 语义搜索基础设施 ──────────────────────────

  题目状态变为 approved/pending_review
       │
       │ 写 embedding_jobs(status=pending)
       ▼
  ┌─────────────────┐    DynamoDB Stream    ┌─────────────────────┐
  │ DynamoDB        │ ─────────────────────► │  Lambda Worker      │
  │ embedding_jobs  │                        │  1. 读 MySQL raw_text│
  └─────────────────┘                        │  2. Bedrock Titan v2│
                                             │     → float64[1536] │
                                             │  3. 写 OpenSearch   │
                                             │  4. job status=done │
                                             └─────────────────────┘
                                                         │
                                                         ▼
                                             ┌─────────────────────┐
                                             │  S3 Vectors         │
                                             │  向量桶              │
                                             │  float64[1536]      │
                                             │  metadata: user_id  │
                                             │           subject   │
                                             └─────────────────────┘
                                                         ▲
                                          语义搜索时      │
                                          QueryVectors───┘

  ─────────────────── 班级协作 ───────────────────────────────────

  ┌──────────────┐  选已有试卷布置  ┌───────────────────────────────┐
  │   试卷编排   │ ───────────────► │           班级任务             │
  │  草稿/发布   │                  │  教师布置 / 学生逐题提交       │
  └──────────────┘                  └──────────┬──────────┬─────────┘
                                               │          │
  ┌──────────┐                      提交触发   ▼          ▼ 提交触发
  │ 班级管理  │ ───────────────────►
  │ 邀请码加入│ 成员资格校验
  └──────────┘

  ─────────────────── 学习闭环 ────────────────────────────────────

                                          ┌───────────────┐
                                          │  复习提交结果  │
                                          │  pass / fail  │
                                          └──────┬────────┘
                                                 │
                              ┌──────────────────┴─────────────────┐
                              │ fail                               │ pass/fail
                              ▼                                    ▼
  ┌──────────┐          ┌──────────┐   ┌──────────┐
  │  错题记录 │ ◄─────── │  任务提交 │   │ 复习调度  │
  │ wrong_cnt │ fail时  │  fail时  │   │Ebbinghaus│
  │ +1        │         └──────────┘   │ interval │
  └────┬─────┘                         └────┬─────┘
       │                                     │
       │  错题摘要                            │  今日待复习
       └──────────────┬──────────────────────┘
                      ▼
               ┌──────────┐
               │  AI 推荐  │
               │  Bedrock  │
               └──────────┘

  ─────────────────── 统计看板 ────────────────────────────────────

  错题记录 + 复习调度(interval_days) + 新增题目数
                  │
                  ▼
         ┌────────────────┐          ┌──────────┐
         │ 统计看板（学生） │          │ 班级管理  │
         │ 错题趋势        │          └────┬─────┘
         │ 科目掌握率      │               │ teacher/admin
         │ 今日待复习数    │               ▼
         └────────────────┘     ┌─────────────────────┐
                                │ 统计看板（班级视图）  │
                                │ 各学生题目数/掌握率   │
                                │ 最后活跃时间          │
                                │ 单学生详情（同学生视图）│
                                └─────────────────────┘
```

---

## 十七、核心业务流程

### 1. 上传识别完整流程

```
  学生         Handler       QuestionSvc    识别API      MySQL       TagSvc
   |              |               |             |            |           |
   |─POST /questions(图片)────────►|             |            |           |
   |              |               |             |            |           |
   |              |─校验格式/大小─►|             |            |           |
   |              | JPEG/PNG      |             |            |           |
   |              | 1KB ~ 10MB    |             |            |           |
   |              |               |             |            |           |
   |              |─Upload()─────►|             |            |           |
   |              |               |─写图片──────────────────►|           |
   |              |               |  /data/imgs/{uid}/{uuid}.jpg         |
   |              |               |             |            |           |
   |              |               |─调识别API──►|            |           |
   |              |               |◄────────────{text,subject,           |
   |              |               |              tags,confidence}        |
   |              |               |             |            |           |
   |              |               | confidence >= 0.85       |           |
   |              |               |─INSERT question(approved)►           |
   |              |               |             |            |           |
   |              |               | 0.50 <= confidence < 0.85|           |
   |              |               |─INSERT question(pending_review)►     |
   |              |               |             |            |           |
   |              |               | confidence < 0.50        |           |
   |              |               |─删除图片文件              |           |
   |              |◄─error────────|             |            |           |
   |◄─422─────────|               |             |            |           |
   |              |               |             |            |           |
   |              |               |─CreateSuggestedTags()───────────────►|
   |              |               |             |            |◄INSERT tags|
   |              |               |             |            | suggested  |
   |              |               |─PUT embedding_jobs──────────────────── DynamoDB
   |              |               |  {question_id,user_id,status=pending}  (失败仅记日志)
   |◄─201/202 JSON|◄─UploadResult─|             |            |           |
```

### 2. 批量上传流程（REQ-UPLOAD-10）

```
POST /api/questions/batch  multipart，含多个图片文件

Handler 遍历每个文件，独立调用 QuestionSvc.Upload()：
  - 成功 → 追加到 succeeded 列表
  - 失败（格式/大小/置信度/识别API错误）→ 追加到 failed 列表，携带原因

整个 batch 不因单文件失败而中止；全部处理完后统一返回：
{
  "succeeded": [{"id": "...", "status": "approved|pending_review"}, ...],
  "failed":    [{"filename": "...", "reason": "..."}, ...],
  "count": N
}
```

### 3. 人工审核 + 通知流程

```
  教师         Handler        ReviewQueueSvc    MySQL      NotifSvc
   |              |                 |              |            |
   |─POST /api/review-queue/submit──►|              |            |
   |              |                 |              |            |
   |              |─Review()───────►|              |            |
   |              |                 |─SELECT question(不过滤user_id)►
   |              |                 |◄─────────────{status=pending_review}
   |              |                 |              |            |
   |              |                 |─UPDATE question──────────►|
   |              |                 |  status=approved/rejected |
   |              |                 |              |            |
   |              |                 |─NotifyQuestionReviewed()─►|
   |              |                 |              |◄─INSERT notification
   |              |◄─nil────────────|              |            |
   |◄─204─────────|                 |              |            |
```

### 3. 复习提交 + Ebbinghaus 调度

```
  学生         Handler       ReviewSvc      MySQL       DynamoDB
   |              |               |             |            |
   |─POST /api/review/submit────────►|             |            |
   |  {question_id, result:pass/fail}|            |            |
   |              |─SubmitResult()►|             |            |
   |              |               |─INSERT review_records────►|
   |              |               |             |            |
   |              |               |─GET schedule(PK=userID,SK=questionID)►
   |              |               |◄────────────────────────{interval_days:N}
   |              |               |             |            |
   |              |               | result=pass  → new_interval = N × 2
   |              |               | result=fail  → new_interval = 1
   |              |               |             |            |
   |              |               |─PUT schedule────────────►|
   |              |               |  {next_review_at,        |
   |              |               |   interval_days, TTL}    |
   |◄─204─────────|◄─nil──────────|             |            |
```

### 4. 班级任务提交流程

```
  学生         Handler       TaskSvc        MySQL       ReviewSvc
   |              |               |             |            |
   |─POST /api/tasks/submit─────────────────────►|            |
   |  {results:[{question_id,result},...]}        |            |
   |              |─Submit()─────►|             |            |
   |              |               |─SELECT task(检查status=active,due_at)►
   |              |               |─SELECT class_members(验证学生在班级)►
   |              |               |             |            |
   |              |               | ┌── 每道题循环 ──────────────────────┐
   |              |               | │─INSERT task_submissions───────────►│
   |              |               | │  UNIQUE冲突 → 409 跳过该题         │
   |              |               | │─SubmitResult()─────────────────────►
   |              |               | │             |◄─INSERT review_records
   |              |               | │             |◄─PUT DynamoDB schedule
   |              |               | └────────────────────────────────────┘
   |◄─207─────────|◄─{succeeded,  |             |            |
   |  Multi-Status|   failed}─────|             |            |
```

---

## 十八、业务规则补充

### 题目删除级联处理（REQ-SEARCH-14）

`POST /api/questions/delete` 的 Service 层执行以下步骤，顺序执行，部分步骤失败不阻断主流程：

```
1. 校验 question.user_id == 当前登录用户（违反 → 403）
2. MySQL 软删除或物理删除 question 记录
3. 删除关联行（同一事务或顺序执行）：
   - question_tags WHERE question_id = ?
   - paper_questions WHERE question_id = ?
   - error_records WHERE question_id = ?
   - review_records WHERE question_id = ?
4. DynamoDB DELETE review_schedules(PK=user_id, SK=question_id)
   失败 → 记录错误日志，不阻断响应
5. DynamoDB DELETE embedding_jobs(PK=question_id, SK="job")
   失败 → 记录错误日志，不阻断响应
6. OpenSearch DELETE question_embeddings WHERE question_id = ?
   失败 → 记录错误日志，不阻断响应
   （kNN 查询强制 filter user_id；孤立向量不会泄露给其他用户）
7. 删除本地图片文件 /data/imgs/{user_id}/{uuid}.jpg
   失败 → 记录错误日志，不阻断响应
8. 返回 204 No Content
```

### question_list / question_search 的角色可见性（REQ-REVIEW-01）

`/api/questions/list` 和 `/api/questions/search` 的 Service 层根据调用者角色自动附加 status 过滤：

| 角色 | 默认可见 status | 说明 |
|------|----------------|------|
| student | approved | pending_review / rejected 对学生不可见 |
| teacher / admin | 全部 status | 可通过 query 参数进一步过滤 |

> student 即使在请求中传入 `status=pending_review` 也无效；Service 层强制覆盖为 `approved`。

### error_records 更新时机

`error_records` 在以下两个场景由对应 Service 层递增更新：

| 触发场景 | 触发 Service | 操作 |
|---------|-------------|------|
| `POST /api/review/submit` 且 result=fail | ReviewSvc | UPSERT error_records：wrong_count+1，last_wrong_at=now() |
| `POST /api/tasks/submit` 且某题 result=fail | TaskSvc（调 ReviewSvc） | 同上，由 ReviewSvc.SubmitResult 统一处理 |

> `result=pass` 时不修改 error_records（只更新 DynamoDB 调度）。

### PDF 文件访问（REQ-PAPER-06）

`POST /api/papers/export` 在生成 PDF 后：
1. 文件写入 `{EXPORT_DIR}/{user_id}/{paper_id}.pdf`
2. 响应返回相对路径 `download_path: "/api/papers/download?id={paper_id}"`
3. `GET /api/papers/download?id={paper_id}` 由后端静态文件路由服务，校验 `paper.user_id == 当前用户` 后以 `Content-Disposition: attachment` 返回文件流

> 新增路由：`GET /api/papers/download  query: id`（需登录）

### 审核操作枚举（REQ-REVIEW-03/04）

`POST /api/review-queue/submit` 的 `action` 字段合法值：

| action | 说明 | 结果 status |
|--------|------|------------|
| `approve` | 审核通过 | approved |
| `reject` | 审核拒绝，`note` 必填 | rejected |

> 其他值 Handler 层返回 400。

### 试卷导出前置校验（REQ-PAPER-02）

`POST /api/papers/export` 在 Handler 层校验：
- `paper_questions` 中该 paper 至少有 1 道题目，否则返回 400

### 标签（REQ-TAG-08 / REQ-TAG-10）

题目详情接口响应中需包含以下字段：
- `has_confirmed_tags` (bool)：当题目无任何 `confirmed` 标签时为 `false`，提示客户端引导学生完成标签确认
- `category_prompt` (bool)：当 `category = "unknown"` 时为 `true`，提示学生手动选择分类；`unknown` 分类不阻止题目加入试卷

### PDF 导出（REQ-PAPER-07）

`papers/export` 在生成 PDF 前对每道题目检查 `status`：
- `status != approved` 的题目从 PDF 中排除
- 若存在被排除的题目，响应中附加 `excluded` 字段，列出排除的 `question_id` 列表，便于前端通知学生

### Bedrock 响应校验（REQ-REC-02 / REQ-REC-03 · CLAUDE.md Rule R2）

`recommend` Service 从 Bedrock 收到响应后，**必须**执行以下 schema 校验，再返回数据：

```
必填字段：question_id (string)、reason (string)
  缺失任一字段 → 整体返回 502，不返回部分列表

可选字段：confidence (float64)
  缺失或非数字 → 默认为 0.0（不视为错误）

confidence 范围：[0.0, 1.0]
  超出范围 → 截断到边界值
```

### 统计（REQ-STATS-04 / REQ-STATS-08）

- `/api/stats/me` 和 `/api/stats/student`：若请求的 `date_from` 至 `date_to` 跨度超过 365 天，截断为从 `date_from` 起的 365 天，并在响应中附加 `"truncated": true`
- `/api/stats/student`：若 `student_id` 对应的用户不存在，或其 `role != student`，返回 404

### 账号管理（REQ-ACCT-02）

CSV 批量导入单次请求行数上限为 200 行；超出时 Handler 层返回 400，不进入业务处理。

### 用户自注销（REQ-ME-05 / REQ-ME-06）

`/api/me/deactivate` 流程：
1. 校验当前密码（Cognito 验证）
2. 更新 `users.status = inactive`，记录 `deactivated_at`
3. 调用 Cognito `GlobalSignOut` 立即吊销所有 session token
4. 自注销账号的 `status` 只能由管理员通过 `/api/admin/users/status` 恢复，用户自身无法重新激活

### 班级与任务（REQ-CLASS-05 / REQ-TASK-08）

- 学生退出班级（`/api/classes/leave`）不删除该学生在班级任务中已提交的 `task_submissions` 记录
- `class_tasks.status` 设为 `closed` 后不能再变为 `active`；Handler 层在 `tasks/update` 接口校验此约束，违反时返回 409

---

## 十九、非功能性目标（REQ-NFR）

| 约束 | 目标值 |
|------|--------|
| API 响应延迟（p95，不含 PDF 生成和识别 API） | ≤ 500 ms |
| PDF 生成（≤100 题） | ≤ 10 s |
| 日志格式 | 结构化 JSON（`log/slog`），每条日志含 `user_id` 字段 |
| 运行时 | Go 1.25 |
| 部署平台 | AWS ECS Fargate |
| 关系型存储 | MySQL（RDS） |
| 调度存储 | DynamoDB（含 TTL） |

---

## 二十、语义搜索（REQ-SEARCH-07 ~ 14）

### 整体数据流

```
题目上传流程（异步写向量）：
  MySQL 写 question
       └─→ DynamoDB 写 embedding_jobs{question_id, user_id, status=pending}
                              │
                       DynamoDB Stream (NEW_AND_OLD_IMAGES)
                              │
                         Lambda Worker (Go runtime)
                              │ 1. MySQL 读 raw_text
                              │ 2. Bedrock Titan Embeddings v2 → float64[1536]
                              │ 3. S3 Vectors PutVectors（含 metadata: user_id, subject）
                              └─→ DynamoDB 更新 status=done

语义搜索流程：
  GET /api/questions/semantic-search?q=...&subject=...
       │
  Handler → SemanticSearchSvc
       │ 1. EmbeddingService.Embed(query) → float64[1536]
       │ 2. S3VectorsRepo.Query(userID, subject, vector, k=5)
       │    → [{question_id, score}, ...]
       │ 3. QuestionRepo.GetByIDs(userID, ids) → []*Question
       └─→ [{question, score}, ...] Top-5，按 score 降序
```

### API

```
GET /api/questions/semantic-search
  query:
    q        string  必填，1~500 字符，自然语言查询文本
    subject  string  可选，科目精确过滤（在 S3 Vectors metadata filter 中应用）
  response 200:
    {"items": [{"question": {...}, "score": 0.92}, ...], "count": N}
  response 400: q 为空或超长
  response 502: Bedrock embedding 服务不可用
```

> 固定返回 Top-5，不分页。与 `/api/questions/search` 独立共存，互不替代。

### 数据模型

**DynamoDB — `embedding_jobs` 表**
```
PK: question_id  String
SK: "job"        String（固定值）
user_id:         String
status:          String   "pending" | "done" | "failed"
retry_count:     Number   默认 0
created_at:      String   ISO8601
TTL:             Number   Unix 时间戳（created_at + 7天）
```

**S3 Vectors — 向量桶**

每条向量记录包含：
```
key:       question_id  (string，全局唯一)
vector:    float64[1536]
metadata:
  user_id:  string   用于查询时强制 filter，保证数据隔离（REQ-SEARCH-13）
  subject:  string   可选 filter，缩小 kNN 比对范围（REQ-SEARCH-09）
```

> 题目内容不冗余存储；向量桶仅存 key + vector + metadata，详情始终从 MySQL 读取。
> S3 Vectors 按 QueryVectors 调用次数计费，无常驻节点，适合中低频语义搜索场景。

### 服务层结构

新增文件：
```
backend/internal/service/embedding.go        EmbeddingService（调 Bedrock Titan）
backend/internal/service/semantic_search.go  SemanticSearchSvc（编排搜索流程）
backend/internal/repository/s3vectors.go     S3VectorsRepo（PutVectors / QueryVectors / DeleteVectors）
lambda/embedding-worker/main.go              Lambda Worker
```

**EmbeddingService**
```go
type EmbeddingService struct {
    client     *bedrockruntime.Client
    modelID    string  // amazon.titan-embed-text-v2:0
    timeoutSec int
}
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float64, error)
```

**SemanticSearchSvc**
```go
func (s *SemanticSearchSvc) Search(ctx context.Context, userID, query, subject string) ([]*SemanticResult, error) {
    // 1. EmbeddingService.Embed(query)
    // 2. S3VectorsRepo.Query(userID, subject, vector, k=5)
    // 3. QuestionRepo.GetByIDs(userID, question_ids)
    // 4. 拼装 SemanticResult{Question, Score}，按 Score 降序
}
```

**Lambda Worker 逻辑**
```
触发：DynamoDB Stream INSERT，status=pending
1. 从 MySQL 读 question.raw_text（按 question_id）
2. 调 Bedrock Titan Embeddings v2 生成向量 float64[1536]
3. S3 Vectors PutVectors:
     key = question_id
     vector = float64[1536]
     metadata = {user_id, subject}
4. 更新 DynamoDB job status=done
失败：status=failed，retry_count+1；retry_count >= 3 时停止重试，记录错误日志
```

### 错误处理

| 场景 | 处理方式 |
|------|---------|
| Bedrock 不可用（搜索时） | 返回 502，不降级为关键词搜索 |
| embedding 尚未生成（上传间隔期） | 该题不出现在结果中，属预期行为 |
| Lambda 重试耗尽（retry_count ≥ 3） | job status=failed，TTL 7天自动清理；该题永不进入语义搜索 |
| 题目删除时 S3 Vectors DeleteVectors 失败 | 记录错误日志，不阻断删除主流程（QueryVectors 强制 filter user_id，孤立向量不泄露） |
| QueryVectors 结果 question_id 已不在 MySQL | GetByIDs 静默跳过，不返回该条目 |

### 新增环境变量

```
S3_VECTORS_BUCKET        S3 Vectors 向量桶名称
EMBEDDING_MODEL_ID       默认 amazon.titan-embed-text-v2:0
EMBEDDING_JOBS_TABLE     DynamoDB embedding_jobs 表名
```

### 依赖变更

```
# 后端新增
github.com/aws/aws-sdk-go-v2/service/s3vectors   S3 Vectors Go 客户端（aws-sdk-go-v2 子包）
```

Lambda Worker 与主服务共用同一 Go module，独立编译为 `lambda/embedding-worker/main.go`。
