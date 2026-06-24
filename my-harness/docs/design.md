# 错题本 — 系统设计文档

**版本**: 1.0
**日期**: 2026-06-24
**技术栈**: Go 1.25 · GORM · MySQL · Vue 3 · AWS ECS Fargate

---

## 一、整体架构

```
[Vue 3 前端 · 静态文件托管]
          │  HTTPS / JSON API
  [Go 1.25 单体服务 · ECS Fargate · :8080]
          │
    ┌─────┼──────────────┐
    │     │              │
 [MySQL  [本地磁盘       [DynamoDB]
  RDS]   /data/imgs]    复习调度]

  [第三方识别 API]   [Amazon Bedrock claude-sonnet-4-6]
```

**关键决策：**
- Go 单体服务，Handler → Service → Repository 三层，无多余抽象
- GORM 作为 MySQL ORM，减少手写 SQL
- DynamoDB 仅用于复习调度（高频读写 + TTL），其余全部在 MySQL
- 图片存储到 ECS Task 挂载的本地卷，路径写入 MySQL
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
```

---

## 四、API 路由

### 认证（公开）
```
POST /api/auth/register
POST /api/auth/login
POST /api/auth/refresh
```

### 用户自管理（需登录）
```
GET    /api/me
PATCH  /api/me/nickname
POST   /api/me/password
DELETE /api/me
```

### 题目（需登录）
```
POST   /api/questions           上传单张
POST   /api/questions/batch     批量上传
GET    /api/questions           列表
GET    /api/questions/search    多维搜索
GET    /api/questions/:id       详情
DELETE /api/questions/:id       删除
PATCH  /api/questions/:id/category  修改分类
```

### 标签（需登录）
```
GET    /api/questions/:id/tags
POST   /api/questions/:id/tags
POST   /api/questions/:id/tags/:tid/confirm
DELETE /api/questions/:id/tags/:tid
```

### 试卷（需登录）
```
POST   /api/papers
GET    /api/papers
GET    /api/papers/:id
PATCH  /api/papers/:id
DELETE /api/papers/:id
POST   /api/papers/:id/questions
DELETE /api/papers/:id/questions/:qid
PUT    /api/papers/:id/reorder
GET    /api/papers/:id/questions
POST   /api/papers/:id/duplicate
POST   /api/papers/:id/export
```

### 复习调度（需登录）
```
GET  /api/review/today
POST /api/review/:id/result
```

### 错题记录（需登录）
```
GET /api/error-records
```

### 通知（需登录）
```
GET  /api/notifications
POST /api/notifications/read-all
POST /api/notifications/:id/read
```

### 统计（需登录）
```
GET /api/stats/me?date_from=&date_to=
GET /api/stats/class?date_from=&date_to=          （teacher/admin）
GET /api/stats/class/:student_id?date_from=&date_to=  （teacher/admin）
```

### AI 推荐（需登录）
```
GET /api/recommend
```

### 导出（需登录）
```
POST /api/export/pdf
```

### 人工审核（teacher/admin）
```
GET  /api/review-queue
POST /api/review-queue/:id/review
```

### 管理员（admin）
```
POST   /api/admin/users
POST   /api/admin/users/import
GET    /api/admin/users
PATCH  /api/admin/users/:id/status
PATCH  /api/admin/users/:id/role
GET    /api/admin/users/:id/questions
```

### 班级（需登录）
```
POST   /api/classes                          （teacher）
GET    /api/classes
GET    /api/classes/:id
DELETE /api/classes/:id/members/:uid         （teacher）
POST   /api/classes/:id/invite-code/reset    （teacher）
POST   /api/classes/join                     （student）
DELETE /api/classes/:id/me                   （student）
```

### 任务（需登录）
```
POST  /api/classes/:id/tasks                 （teacher）
GET   /api/classes/:id/tasks
PATCH /api/classes/:id/tasks/:tid            （teacher）
GET   /api/classes/:id/tasks/:tid/progress   （teacher）
GET   /api/classes/:id/tasks/:tid
POST  /api/classes/:id/tasks/:tid/submit     （student）
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
- 封装所有数据库操作
- 所有查询强制带 `user_id` 条件（REQ-SEC-01）
- 不含业务逻辑

> **注**：Repository 层不单独建包，直接在 Service 中通过 `db *gorm.DB` 操作，保持代码简单。大查询方法独立为函数即可。

---

## 六、认证方案

使用 Amazon Cognito User Pool：
- 注册/登录调 Cognito SDK，返回 JWT（AccessToken + RefreshToken）
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
```

---

## 八、Ebbinghaus 复习调度

```
提交复习结果（pass/fail）：
  pass → interval_days = 上次 interval_days × 2（首次为 1）
  fail → interval_days = 1
  next_review_at = now() + interval_days
  写入 DynamoDB（PK=user_id, SK=question_id），覆盖旧记录
  TTL = next_review_at + 30天

查询今日待复习：
  DynamoDB Query: PK=user_id, next_review_at ≤ today
```

---

## 九、科目掌握率计算

```
对每道 approved 题目取 DynamoDB 中的 interval_days：
  weight = log2(interval_days + 1)
  max_weight = log2(65)  // 上限 64 天

mastery_rate(科目) = Σweight / (题目数 × max_weight)
结果截断到 [0.0, 1.0]，保留两位小数
```

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
```

---

## 十二、前端结构（Vue 3）

```
src/
  api/
    auth.js         登录/注册/刷新
    question.js     题目 CRUD、上传、搜索
    paper.js        试卷编排、导出
    review.js       复习调度
    stats.js        统计看板
    class.js        班级、任务
    admin.js        管理员接口
  views/
    Login.vue
    Register.vue
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
github.com/aws/aws-sdk-go-v2        Cognito / Bedrock / DynamoDB
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

## 十四、模块交互图

```mermaid
flowchart TD
    subgraph 前端["前端 Vue 3"]
        UI[用户浏览器]
    end

    subgraph 后端["后端 Go 单体服务"]
        MW["Middleware\nJWT校验 · 角色检查"]

        subgraph Handlers["Handler 层"]
            H_AUTH[auth]
            H_Q[question]
            H_TAG[tag]
            H_PAPER[paper]
            H_REVIEW[review]
            H_REC[recommend]
            H_NOTIF[notification]
            H_STATS[stats]
            H_CLASS[class]
            H_TASK[task]
            H_ADMIN[admin]
            H_ME[me]
        end

        subgraph Services["Service 层"]
            S_AUTH[auth]
            S_Q[question]
            S_TAG[tag]
            S_PAPER[paper]
            S_REVIEW[review]
            S_REC[recommend]
            S_NOTIF[notification]
            S_STATS[stats]
            S_CLASS[class]
            S_TASK[task]
            S_ADMIN[admin]
        end
    end

    subgraph 存储["存储层"]
        MYSQL[(MySQL\nRDS)]
        DYNAMO[(DynamoDB\n复习调度)]
        DISK[/本地磁盘\n/data/imgs/]
    end

    subgraph 外部["外部服务"]
        COGNITO[Amazon Cognito\n认证]
        RECOG[第三方识别 API\n图片→题目文本]
        BEDROCK[Amazon Bedrock\nclaude-sonnet-4-6]
    end

    UI -->|HTTPS JSON| MW
    MW --> H_AUTH & H_Q & H_TAG & H_PAPER
    MW --> H_REVIEW & H_REC & H_NOTIF & H_STATS
    MW --> H_CLASS & H_TASK & H_ADMIN & H_ME

    H_AUTH --> S_AUTH
    H_Q --> S_Q
    H_TAG --> S_TAG
    H_PAPER --> S_PAPER
    H_REVIEW --> S_REVIEW
    H_REC --> S_REC
    H_NOTIF --> S_NOTIF
    H_STATS --> S_STATS
    H_CLASS --> S_CLASS
    H_TASK --> S_TASK
    H_ADMIN --> S_ADMIN
    H_ME --> S_ADMIN

    S_AUTH -->|创建/禁用账号| COGNITO
    S_Q -->|写题目| MYSQL
    S_Q -->|存图片| DISK
    S_Q -->|识别请求| RECOG
    S_TAG --> MYSQL
    S_PAPER --> MYSQL
    S_REVIEW -->|写复习记录| MYSQL
    S_REVIEW -->|读写调度| DYNAMO
    S_REC -->|错题摘要| BEDROCK
    S_NOTIF --> MYSQL
    S_STATS --> MYSQL
    S_STATS -->|读调度interval| DYNAMO
    S_CLASS --> MYSQL
    S_TASK --> MYSQL
    S_TASK -->|触发调度更新| S_REVIEW
    S_ADMIN -->|禁用账号| COGNITO
    S_ADMIN --> MYSQL

    %% 跨 Service 调用
    S_Q -->|上传后创建suggested标签| S_TAG
    S_REVIEW -->|审核完成发通知| S_NOTIF
```

---

## 十五、数据库 ER 图

```mermaid
erDiagram
    users {
        varchar36 id PK
        varchar128 cognito_sub UK
        varchar255 email UK
        varchar100 nickname
        enum role "student|teacher|admin"
        enum status "active|inactive"
        datetime deactivated_at
        datetime created_at
    }

    questions {
        varchar36 id PK
        varchar36 user_id FK
        varchar500 image_path
        text raw_text
        varchar100 subject
        enum category "multiple_choice|fill_blank|essay|true_false|calculation|unknown"
        enum status "pending_review|approved|rejected"
        decimal confidence
        text review_note
        varchar36 reviewed_by FK
        datetime reviewed_at
        datetime created_at
    }

    question_tags {
        varchar36 id PK
        varchar36 question_id FK
        varchar36 user_id FK
        varchar100 name
        enum status "suggested|confirmed"
        datetime created_at
    }

    papers {
        varchar36 id PK
        varchar36 user_id FK
        varchar200 title
        enum status "draft|published"
        datetime created_at
        datetime updated_at
    }

    paper_questions {
        varchar36 id PK
        varchar36 paper_id FK
        varchar36 question_id FK
        int position
        datetime created_at
    }

    error_records {
        varchar36 id PK
        varchar36 user_id FK
        varchar36 question_id FK
        int wrong_count
        datetime last_wrong_at
        datetime created_at
    }

    review_records {
        varchar36 id PK
        varchar36 user_id FK
        varchar36 question_id FK
        enum result "pass|fail"
        datetime reviewed_at
    }

    notifications {
        varchar36 id PK
        varchar36 user_id FK
        varchar50 type
        varchar200 title
        text body
        varchar36 ref_id
        bool is_read
        datetime created_at
    }

    classes {
        varchar36 id PK
        varchar100 name
        varchar36 teacher_id FK
        varchar6 invite_code UK
        datetime created_at
    }

    class_members {
        varchar36 id PK
        varchar36 class_id FK
        varchar36 user_id FK
        datetime joined_at
    }

    class_tasks {
        varchar36 id PK
        varchar36 class_id FK
        varchar36 paper_id FK
        varchar200 title
        varchar36 assigned_by FK
        datetime due_at
        enum status "active|closed"
        datetime created_at
    }

    task_submissions {
        varchar36 id PK
        varchar36 task_id FK
        varchar36 user_id FK
        varchar36 question_id FK
        enum result "pass|fail"
        datetime submitted_at
    }

    users ||--o{ questions : "拥有"
    users ||--o{ question_tags : "拥有"
    users ||--o{ papers : "拥有"
    users ||--o{ error_records : "产生"
    users ||--o{ review_records : "提交"
    users ||--o{ notifications : "接收"
    users ||--o{ class_members : "加入"
    users ||--o{ task_submissions : "提交"
    users ||--o{ classes : "教授"

    questions ||--o{ question_tags : "拥有"
    questions ||--o{ paper_questions : "包含于"
    questions ||--o{ error_records : "关联"
    questions ||--o{ review_records : "关联"
    questions ||--o{ task_submissions : "关联"

    papers ||--o{ paper_questions : "包含"
    papers ||--o{ class_tasks : "用于"

    classes ||--o{ class_members : "包含"
    classes ||--o{ class_tasks : "发布"

    class_tasks ||--o{ task_submissions : "收到"
```

---

## 十六、功能模块关系图

> 箭头表示依赖方向（A → B 表示 A 依赖 B 提供的能力）。
> 实线为强依赖（核心流程必须），虚线为弱依赖（触发或通知）。

```mermaid
flowchart LR
    subgraph 用户体系
        AUTH[认证模块\nCognito JWT]
        ACCT[账号管理\n管理员创建/停用]
        ME[用户自管理\n改密码/昵称/注销]
    end

    subgraph 题目核心
        UPLOAD[上传识别\n图片→题目]
        TAG[标签管理\nsuggest/confirm]
        SEARCH[搜索\n多维过滤]
        QREVIEW[人工审核\n审核队列]
    end

    subgraph 学习闭环
        SCHED[复习调度\nEbbinghaus]
        EREC[错题记录\nwrong_count]
        REC[AI 推荐\nBedrock]
    end

    subgraph 组卷导出
        PAPER[试卷编排\n草稿/发布]
        EXPORT[PDF 导出\ngofpdf]
    end

    subgraph 协作
        CLASS[班级管理\n邀请码]
        TASK[班级任务\n布置/提交]
    end

    subgraph 辅助
        NOTIF[通知\n站内消息]
        STATS[统计看板\n趋势/掌握率]
    end

    %% 所有模块都依赖认证
    AUTH -.->|JWT userID| UPLOAD & TAG & SEARCH & QREVIEW
    AUTH -.->|JWT userID| SCHED & EREC & REC & PAPER & EXPORT
    AUTH -.->|JWT userID| CLASS & TASK & NOTIF & STATS & ME & ACCT

    %% 账号管理
    ACCT -->|调 Cognito| AUTH
    ME -->|调 Cognito 改密码| AUTH

    %% 题目核心流程
    UPLOAD -->|置信度分级后入库| QREVIEW
    UPLOAD -->|识别结果生成| TAG
    QREVIEW -->|审核完成触发| NOTIF
    TAG -->|confirmed 标签供| SEARCH
    SEARCH -->|同一搜索条件| PAPER

    %% 学习闭环
    EREC -->|错题摘要输入| REC
    SCHED -->|今日待复习列表| REC
    SCHED -->|调度更新| EREC

    %% 组卷导出
    PAPER -->|只含 approved 题| EXPORT
    PAPER -->|发布为班级任务| TASK

    %% 协作
    CLASS -->|成员资格校验| TASK
    TASK -->|提交结果触发| SCHED
    TASK -->|提交结果触发| EREC

    %% 统计
    SCHED -->|interval_days| STATS
    EREC -->|wrong_count| STATS
    UPLOAD -->|新增题目数| STATS
```

---

## 十七、核心业务流程

### 1. 上传识别完整流程

```mermaid
sequenceDiagram
    participant U as 学生
    participant H as Handler
    participant S as QuestionService
    participant R as 识别API
    participant DB as MySQL
    participant TS as TagService

    U->>H: POST /api/questions (图片)
    H->>H: 校验格式/大小 (JPEG/PNG, 1KB~10MB)
    H->>S: Upload(userID, imageBytes)
    S->>S: 写图片到 /data/imgs/{userID}/{uuid}.jpg
    S->>R: 调识别API(base64图片)
    R-->>S: {raw_text, subject, tags, confidence}
    alt confidence >= 0.85
        S->>DB: INSERT question (status=approved)
    else 0.50 <= confidence < 0.85
        S->>DB: INSERT question (status=pending_review)
    else confidence < 0.50
        S->>S: 删除图片文件
        S-->>H: 返回 422 错误
    end
    S->>TS: CreateSuggestedTags(questionID, tags)
    TS->>DB: INSERT question_tags (status=suggested)
    S-->>H: UploadResult
    H-->>U: 201/202 + question JSON
```

### 2. 人工审核 + 通知流程

```mermaid
sequenceDiagram
    participant T as 教师
    participant H as ReviewQueueHandler
    participant S as ReviewQueueService
    participant DB as MySQL
    participant NS as NotificationService

    T->>H: POST /api/review-queue/:id/review
    H->>S: Review(questionID, reviewerID, action)
    S->>DB: SELECT question WHERE id=? (不过滤user_id)
    DB-->>S: question (status=pending_review)
    S->>DB: UPDATE question SET status=approved/rejected
    S->>NS: NotifyQuestionReviewed(question.user_id, status)
    NS->>DB: INSERT notifications
    S-->>H: nil
    H-->>T: 204 No Content
```

### 3. 复习提交 + Ebbinghaus 调度

```mermaid
sequenceDiagram
    participant U as 学生
    participant H as ReviewHandler
    participant S as ReviewService
    participant DB as MySQL
    participant DDB as DynamoDB

    U->>H: POST /api/review/:id/result {result: pass/fail}
    H->>S: SubmitResult(userID, questionID, result)
    S->>DB: INSERT review_records
    S->>DDB: GET schedule (PK=userID, SK=questionID)
    DDB-->>S: {interval_days: N}
    alt result = pass
        S->>S: new_interval = N * 2
    else result = fail
        S->>S: new_interval = 1
    end
    S->>DDB: PUT schedule {next_review_at, interval_days, TTL}
    S-->>H: nil
    H-->>U: 204 No Content
```

### 4. 班级任务提交流程

```mermaid
sequenceDiagram
    participant U as 学生
    participant H as TaskHandler
    participant S as TaskService
    participant DB as MySQL
    participant RS as ReviewService

    U->>H: POST /api/classes/:id/tasks/:tid/submit
    H->>S: Submit(taskID, userID, results[])
    S->>DB: SELECT task (检查 status=active, due_at)
    S->>DB: SELECT class_members (验证学生在班级)
    loop 每道题
        S->>DB: INSERT task_submissions (UNIQUE冲突→409)
        S->>RS: SubmitResult(userID, questionID, result)
        RS->>DB: INSERT review_records
        RS->>DB: UPDATE DynamoDB schedule
    end
    S-->>H: {succeeded, failed}
    H-->>U: 207 Multi-Status
```
