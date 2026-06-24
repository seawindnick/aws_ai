# 班级协作 设计文档

**日期**: 2026-06-24
**状态**: 已确认，待实现
**关联需求**: REQ-CLASS-01~07, REQ-TASK-01~08
**依赖**: Spec A（账号体系，teacher/student 角色）, REQ-SCHED-01（复习调度）

---

## 一、概述

本模块覆盖两个紧密关联的子功能：

1. **班级管理** — 教师创建班级并生成邀请码，学生用邀请码自助加入；教师可管理成员，学生可退出
2. **任务布置与提交** — 教师把自己的试卷布置为班级任务（可设截止日期），学生逐题提交 pass/fail，结果同步触发个人复习调度，教师实时查看进度

---

## 二、数据模型

```sql
-- 班级
CREATE TABLE classes (
  id          VARCHAR(36)  NOT NULL PRIMARY KEY,
  name        VARCHAR(100) NOT NULL,
  teacher_id  VARCHAR(36)  NOT NULL,
  invite_code VARCHAR(6)   NOT NULL UNIQUE,
  created_at  DATETIME     NOT NULL,
  INDEX idx_teacher (teacher_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 班级成员
CREATE TABLE class_members (
  id         VARCHAR(36) NOT NULL PRIMARY KEY,
  class_id   VARCHAR(36) NOT NULL,
  user_id    VARCHAR(36) NOT NULL,
  joined_at  DATETIME    NOT NULL,
  UNIQUE KEY uq_class_user (class_id, user_id),
  INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 班级任务
CREATE TABLE class_tasks (
  id           VARCHAR(36)  NOT NULL PRIMARY KEY,
  class_id     VARCHAR(36)  NOT NULL,
  paper_id     VARCHAR(36)  NOT NULL,
  title        VARCHAR(200) NOT NULL,
  assigned_by  VARCHAR(36)  NOT NULL,
  due_at       DATETIME     NULL,
  status       ENUM('active','closed') NOT NULL DEFAULT 'active',
  created_at   DATETIME     NOT NULL,
  INDEX idx_class (class_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 学生提交记录
CREATE TABLE task_submissions (
  id           VARCHAR(36) NOT NULL PRIMARY KEY,
  task_id      VARCHAR(36) NOT NULL,
  user_id      VARCHAR(36) NOT NULL,
  question_id  VARCHAR(36) NOT NULL,
  result       ENUM('pass','fail') NOT NULL,
  submitted_at DATETIME    NOT NULL,
  UNIQUE KEY uq_task_user_question (task_id, user_id, question_id),
  INDEX idx_task_user (task_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 三、API 设计

### 教师端（需 teacher 或 admin 角色）

```
POST   /api/classes                              创建班级
GET    /api/classes                              我教的班级列表
GET    /api/classes/{id}                         班级详情 + 成员列表
DELETE /api/classes/{id}/members/{uid}           移除学生
POST   /api/classes/{id}/invite-code/reset       重置邀请码

POST   /api/classes/{id}/tasks                   布置任务
GET    /api/classes/{id}/tasks                   班级任务列表
PATCH  /api/classes/{id}/tasks/{tid}             更新 due_at 或关闭任务
GET    /api/classes/{id}/tasks/{tid}/progress    任务完成进度
```

### 学生端

```
POST   /api/classes/join                         用邀请码加入班级
GET    /api/classes                              我加入的班级列表
DELETE /api/classes/{id}/me                      退出班级

GET    /api/classes/{id}/tasks                   我的任务列表
GET    /api/classes/{id}/tasks/{tid}             任务详情（题目 + 我的提交状态）
POST   /api/classes/{id}/tasks/{tid}/submit      提交题目结果
```

---

## 四、关键行为说明

### 邀请码

- 6 位大写字母+数字，系统随机生成，全局唯一（`classes.invite_code UNIQUE`）
- 教师调用 `POST /api/classes/{id}/invite-code/reset` 后旧码立即失效（UPDATE 覆盖）
- 学生加入时按邀请码查找班级，找不到返回 404，不区分"无效"和"已过期"

### 任务布置

- `paper_id` 必须归属于发起请求的 teacher（`papers.user_id = teacher_id`），否则返回 403
- 试卷可为 draft 或 published 状态，均可布置（任务发布不改变试卷状态）
- `due_at` 可为空，空时任务长期有效

### 学生提交

提交流程（在同一事务或顺序操作中）：
1. 验证任务存在且学生是该班级成员
2. 检查 `due_at`：若不为空且已过期，返回 403
3. 检查重复提交：`(task_id, user_id, question_id)` 已存在返回 409
4. 写入 `task_submissions`
5. 触发复习调度更新（调用 ReviewService.SubmitResult，与 REQ-SCHED-01 相同逻辑）

步骤 5 失败不回滚步骤 4，但返回 error 让调用方知晓调度未更新（R1 不吞 error）。

### 批量提交

`POST /api/classes/{id}/tasks/{tid}/submit` 支持一次提交多道题：

```json
{
  "results": [
    {"question_id": "...", "result": "pass"},
    {"question_id": "...", "result": "fail"}
  ]
}
```

每条独立处理，部分失败时响应体列出失败条目，HTTP 状态为 207 Multi-Status。

### 任务进度

`GET /api/classes/{id}/tasks/{tid}/progress` 响应：

```json
{
  "total_students": 32,
  "total_questions": 10,
  "per_student": [
    {
      "user_id": "...",
      "nickname": "张三",
      "submitted_count": 7,
      "total_count": 10,
      "pass_count": 5
    }
  ]
}
```

仅班级内的 teacher（`classes.teacher_id`）可访问，其他用户返回 403。

### 任务关闭

`PATCH /api/classes/{id}/tasks/{tid}` body：
- `{due_at: "2026-07-01T23:59:59Z"}` — 更新截止时间
- `{status: "closed"}` — 提前关闭，关闭后不可再提交，不可重新开启

---

## 五、权限矩阵

| 操作 | student | teacher（本班） | teacher（他班） | admin |
|------|---------|----------------|----------------|-------|
| 创建班级 | ✗ 403 | ✓ | ✓ | ✓ |
| 查看班级详情 | ✓（已加入） | ✓（自己的） | ✗ 403 | ✓ |
| 移除成员 | ✗ 403 | ✓ | ✗ 403 | ✓ |
| 布置任务 | ✗ 403 | ✓（自己的班） | ✗ 403 | ✓ |
| 查看进度 | ✗ 403 | ✓（自己的班） | ✗ 403 | ✓ |
| 提交结果 | ✓（已加入） | ✗ 403 | ✗ 403 | ✗ 403 |

---

## 六、分层约束

遵循 Handler → Service → Repository 三层规则（R6）：

- `handler/class.go`：班级 CRUD、邀请码重置、成员管理
- `handler/task.go`：任务布置、提交、进度查询
- `service/class.go`：邀请码唯一性保障、成员资格校验
- `service/task.go`：提交流程编排（写 submission → 触发调度）
- `repository/class.go`：classes、class_members 表操作
- `repository/task.go`：class_tasks、task_submissions 表操作

---

## 七、错误处理

| 场景 | HTTP 状态 |
|------|-----------|
| 邀请码无效或已失效 | 404 |
| 学生已在班级中 | 409 |
| 任务已过截止日期，仍提交 | 403 |
| 重复提交同一题目 | 409 |
| 布置任务时 paper 不属于该教师 | 403 |
| 非班级成员访问班级资源 | 403 |
| 任务已关闭，仍提交 | 403 |

---

## 八、EARS 需求条目

### 班级管理

**REQ-CLASS-01**
The system shall allow a teacher to create a named class; the system shall generate a unique 6-character alphanumeric invite code for the class.

**REQ-CLASS-02**
The system shall allow a student to join a class by submitting a valid invite code; a student may belong to multiple classes simultaneously.

**REQ-CLASS-03**
When a teacher resets a class invite code, the system shall generate a new unique code and immediately invalidate the previous code.

**REQ-CLASS-04**
The system shall allow a teacher to view the member list of their own class and remove any student from it.

**REQ-CLASS-05**
The system shall allow a student to leave a class voluntarily; leaving a class shall not delete the student's submitted task results.

**REQ-CLASS-06**
If a user attempts to join a class with an invalid or expired invite code, then the system shall return a 404 Not Found response.

**REQ-CLASS-07**
If a student attempts to create a class, delete a member, or reset an invite code, then the system shall return a 403 Forbidden response.

### 任务管理

**REQ-TASK-01**
The system shall allow a teacher to assign a review task to a class by selecting one of their own papers, with an optional due date.

**REQ-TASK-02**
The system shall allow a student to view the task list for each of their classes, including each task's title, due date, and their own completion status.

**REQ-TASK-03**
The system shall allow a student to submit pass or fail results for each question in a task; each (task, student, question) combination shall only be submitted once.

**REQ-TASK-04**
When a student submits a task question result, the system shall update the student's personal Ebbinghaus review schedule for that question, following the same rules as REQ-SCHED-01.

**REQ-TASK-05**
If a student attempts to submit results for a task that is closed or whose due date has passed, then the system shall return a 403 Forbidden response.

**REQ-TASK-06**
If a student attempts to submit results for the same question in the same task more than once, then the system shall return a 409 Conflict response.

**REQ-TASK-07**
The system shall allow a teacher to view per-student completion progress for any task in their class, including submitted count, total question count, and pass count per student.

**REQ-TASK-08**
The system shall allow a teacher to update the due date of an active task or close it early; a closed task shall not accept further student submissions and shall not be reopened.
