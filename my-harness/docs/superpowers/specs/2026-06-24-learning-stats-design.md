# 学习统计看板 设计文档

**日期**: 2026-06-24
**状态**: 已确认，待实现
**关联需求**: REQ-STATS-01~08

---

## 一、概述

本模块提供两类统计视图：

1. **学生自查** — 查看自己在自定义日期范围内的错题趋势、科目掌握率、今日待复习数
2. **教师查班** — 查看全班学生摘要列表，可下钻到单个学生的完整统计详情

所有数据均为只读查询，不写入任何业务表。掌握率计算基于 DynamoDB 复习调度中的 `interval_days`。

---

## 二、API 设计

### 学生端

```
GET /api/stats/me?date_from=YYYY-MM-DD&date_to=YYYY-MM-DD
```

### 教师 / 管理员端

```
GET /api/stats/class?date_from=YYYY-MM-DD&date_to=YYYY-MM-DD    班级学生摘要列表
GET /api/stats/class/{student_id}?date_from=&date_to=           单个学生完整统计
```

两端 `date_from` / `date_to` 均为必填，最大跨度 365 天，超出则截断至 `date_from + 365 天` 并在响应中返回 `truncated: true`。

---

## 三、响应结构

### `/api/stats/me`

```json
{
  "date_from": "2026-01-01",
  "date_to": "2026-06-24",
  "truncated": false,
  "error_trend": [
    {"date": "2026-06-01", "count": 3},
    {"date": "2026-06-02", "count": 1}
  ],
  "subject_mastery": [
    {"subject": "数学", "mastery_rate": 0.72, "question_count": 18},
    {"subject": "英语", "mastery_rate": 0.45, "question_count": 11}
  ],
  "today_pending_count": 5
}
```

### `/api/stats/class`

```json
{
  "items": [
    {
      "user_id": "...",
      "nickname": "张三",
      "email": "zhang@school.edu",
      "question_count": 24,
      "avg_mastery_rate": 0.61,
      "today_pending_count": 3,
      "last_active_at": "2026-06-23"
    }
  ],
  "count": 42
}
```

### `/api/stats/class/{student_id}`

结构与 `/api/stats/me` 完全一致，额外包含 `user_id` 字段。

---

## 四、核心计算逻辑

### 错题趋势

查询 `questions` 表，按 `DATE(created_at)` 分组统计 `COUNT(*)`，条件：
- `user_id = ?`（R4 数据隔离）
- `created_at BETWEEN date_from AND date_to + 1 day`

日期范围内无数据的日期不补零（由前端决定展示方式）。

### 科目掌握率（综合加权）

对每道 `approved` 题目取 DynamoDB 中最新的 `interval_days`：

```
weight(q) = log2(interval_days + 1)
max_weight = log2(理论最大 interval_days + 1)  // 当前取 64 天上限，即 max_weight = log2(65) ≈ 6
mastery_rate = Σ weight(q) / (question_count × max_weight)
```

- 无复习记录的题目：`interval_days = 0`，`weight = 0`
- 结果截断到 [0.0, 1.0]，保留两位小数
- 按科目分组，`subject` 来自 `questions.subject`

### 今日待复习数

查询 DynamoDB `review_schedule`，`user_id = ?` + `next_review_at ≤ now()`，返回记录条数。不返回题目列表（列表由 `GET /api/review/today` 负责）。

### 教师班级摘要

`avg_mastery_rate` = 该学生所有科目掌握率的算术平均值（无数据时为 0）。  
`last_active_at` = `MAX(questions.created_at, review_records.reviewed_at)` 取较晚者的日期部分。

---

## 五、权限控制

| 接口 | student | teacher | admin |
|------|---------|---------|-------|
| `GET /api/stats/me` | ✓ | ✓（查自己） | ✓（查自己） |
| `GET /api/stats/class` | ✗ 403 | ✓ | ✓ |
| `GET /api/stats/class/{id}` | ✗ 403 | ✓ | ✓ |

教师查询 `{student_id}` 时，若该 user 不存在或 role 不为 student，返回 404。

---

## 六、分层约束

遵循 Handler → Service → Repository 三层规则（R6）：

- `handler/stats.go`：参数解析、日期校验、截断逻辑
- `service/stats.go`：掌握率计算、聚合逻辑、DynamoDB 查询编排
- `repository/stats.go`：MySQL 聚合查询（错题趋势、科目分组）；DynamoDB 查询复用 `ReviewRepo`

---

## 七、错误处理

| 场景 | HTTP 状态 |
|------|-----------|
| 缺少 date_from 或 date_to | 400 |
| 日期格式错误（非 YYYY-MM-DD） | 400 |
| 日期范围超 365 天 | 200，含 `truncated: true` |
| student 访问 `/api/stats/class/*` | 403 |
| 查询不存在或非 student 的用户 | 404 |

---

## 八、EARS 需求条目

**REQ-STATS-01**
The system shall allow a student to retrieve their learning statistics for a custom date range (maximum span: 365 days), including daily new question counts, per-subject mastery rates, and today's pending review count.

**REQ-STATS-02**
The system shall compute per-subject mastery rate using an interval-weighted formula: each question's weight is `log2(current_interval_days + 1)`; questions with no review history contribute weight 0; the result is normalized to the range [0.0, 1.0].

**REQ-STATS-03**
The system shall return today's pending review count as a single integer derived from the review schedule, without returning the full question list.

**REQ-STATS-04**
If the requested date range exceeds 365 days, then the system shall truncate to 365 days from `date_from` and include a `truncated: true` flag in the response.

**REQ-STATS-05**
The system shall allow a teacher or admin to retrieve a summary list of all students' statistics for a given date range, including question count, average mastery rate, today's pending review count, and last active date per student.

**REQ-STATS-06**
The system shall allow a teacher or admin to retrieve the full statistics detail of any individual student, using the same structure as the student's own statistics response.

**REQ-STATS-07**
If a student attempts to access `/api/stats/class/*`, then the system shall return a 403 Forbidden response.

**REQ-STATS-08**
If a teacher or admin requests statistics for a user who does not exist or is not a student, then the system shall return a 404 Not Found response.
