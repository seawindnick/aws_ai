# 账号与用户管理 设计文档

**日期**: 2026-06-24
**状态**: 已确认，待实现
**关联需求**: REQ-ACCT-01~08, REQ-ME-01~06

---

## 一、概述

本模块覆盖两个独立但共用同一数据模型的子功能：

1. **管理员账号管理** — 创建、批量导入、停用/恢复、角色变更、查看任意用户数据
2. **用户自管理** — 查看个人信息、改昵称、改密码、注销账号

认证体系基于 Amazon Cognito，账号状态在 Cognito 和本地 `users` 表双写保持同步。

---

## 二、数据模型变更

在现有 `users` 表基础上新增三列：

```sql
ALTER TABLE users
  ADD COLUMN nickname      VARCHAR(100)  NULL,
  ADD COLUMN status        ENUM('active','inactive') NOT NULL DEFAULT 'active',
  ADD COLUMN deactivated_at DATETIME     NULL;
```

- `nickname`：用户显示名，可为空（空时前端回退显示 email 前缀）
- `status`：账号状态，停用或注销后置为 `inactive`，Cognito 侧同步禁用
- `deactivated_at`：停用/注销时间戳，用于审计

---

## 三、API 设计

### 管理员端（需 admin 角色）

```
POST   /api/admin/users                  单个创建账号
POST   /api/admin/users/import           批量 CSV 导入
GET    /api/admin/users                  用户列表（?role=&status=&page=&page_size=）
PATCH  /api/admin/users/{id}/status      停用 / 恢复（body: {status: "inactive"|"active"}）
PATCH  /api/admin/users/{id}/role        变更角色（body: {role: "student"|"teacher"|"admin"}）
GET    /api/admin/users/{id}/questions   查看任意用户题目列表（只读）
```

### 用户自管理端（任意已认证用户）

```
GET    /api/me                           查看个人信息
PATCH  /api/me/nickname                  修改昵称（body: {nickname: "..."}）
POST   /api/me/password                  修改密码（body: {old_password, new_password}）
DELETE /api/me                           注销账号（body: {password} 二次确认）
```

---

## 四、关键行为说明

### 账号创建

- 系统生成 8 位随机初始密码（字母+数字），在创建响应的 `initial_password` 字段中返回一次，此后不再可查
- 同时在 Cognito User Pool 和本地 `users` 表各创建一条记录
- Cognito 侧标记为 `FORCE_CHANGE_PASSWORD` 状态，用户首次登录后须修改密码

### 批量导入

- CSV 格式：`email,role`，第一行为标题行，跳过
- 每行独立处理，单行失败不中止后续行
- 响应格式：`{succeeded: [{email, initial_password}], failed: [{row, email, reason}]}`
- 单次导入上限 200 行

### 停用账号

1. Cognito：`AdminDisableUser` → 立即使所有已签发 token 失效
2. 本地：`users.status = 'inactive'`, `deactivated_at = now()`
3. 数据保留：题目、复习记录、错题记录、试卷全部保留
4. 管理员可通过 `GET /api/admin/users/{id}/questions` 只读访问停用用户数据

### 恢复账号

1. Cognito：`AdminEnableUser`
2. 本地：`users.status = 'active'`, `deactivated_at = NULL`

### 用户自注销

1. 验证当前密码（调 Cognito `InitiateAuth` 确认）
2. Cognito：`AdminDisableUser`
3. 本地：`users.status = 'inactive'`, `deactivated_at = now()`
4. 自注销后无法自助恢复，需管理员介入

### 改密码

- 调用 Cognito `ChangePassword`（需携带当前 AccessToken）
- 新密码规则：8–72 字符，至少含一个字母和一个数字
- 旧密码错误由 Cognito 返回，系统映射为 400 Bad Request

---

## 五、分层约束

遵循现有 Handler → Service → Repository 三层规则（R6）：

- `handler/admin_user.go`：输入校验（email 格式、role 枚举、CSV 解析）
- `service/admin_user.go`：Cognito 调用 + 本地 DB 写入，双写失败时返回 error（不静默降级，R2）
- `repository/user.go`：扩展现有 UserRepo，新增 `UpdateStatus`、`UpdateRole`、`UpdateNickname`
- `handler/me.go`：用户自管理接口，复用 UserRepo

---

## 六、错误处理

| 场景 | HTTP 状态 | 响应 |
|------|-----------|------|
| 非 admin 访问 `/api/admin/*` | 403 | `{"error": "forbidden"}` |
| 创建账号时 email 已存在 | 409 | `{"error": "email already registered"}` |
| CSV 行格式错误 | 包含在 `failed[]` 中 | 不影响整批 |
| 改密码旧密码错误 | 400 | `{"error": "current password incorrect"}` |
| 注销时密码验证失败 | 400 | `{"error": "password confirmation failed"}` |
| Cognito 服务不可用 | 502 | `{"error": "auth service unavailable"}` |

---

## 七、EARS 需求条目

### 管理员账号管理

**REQ-ACCT-01**
The system shall allow an admin to create a user account by specifying an email address and role; the system shall generate a random initial password and return it in the creation response, displayed only once.

**REQ-ACCT-02**
The system shall allow an admin to import user accounts in bulk by uploading a CSV file containing email and role columns; each row shall be processed independently and the response shall list succeeded and failed rows without aborting the entire batch. A single import request shall not exceed 200 rows.

**REQ-ACCT-03**
When an admin deactivates a user account, the system shall immediately revoke the user's ability to authenticate and shall retain all of that user's question and review data.

**REQ-ACCT-04**
When an admin reactivates a deactivated account, the system shall restore the user's ability to authenticate and all previously retained data shall remain accessible.

**REQ-ACCT-05**
The system shall allow an admin to change a user's role to student, teacher, or admin.

**REQ-ACCT-06**
The system shall allow an admin to list all user accounts with filtering by role and status, with pagination (default page size: 20, maximum: 100).

**REQ-ACCT-07**
The system shall allow an admin to view the question list of any user, including deactivated users, in read-only mode.

**REQ-ACCT-08**
If a non-admin user attempts to access any `/api/admin/*` endpoint, then the system shall return a 403 Forbidden response.

### 用户自管理

**REQ-ME-01**
The system shall allow an authenticated user to retrieve their own profile (id, email, nickname, role, created_at).

**REQ-ME-02**
The system shall allow an authenticated user to update their nickname (1–50 characters).

**REQ-ME-03**
When an authenticated user submits a password change request with their current password and a new password (8–72 characters, at least one letter and one digit), the system shall update the credential via Cognito and return 204 No Content.

**REQ-ME-04**
If the current password provided during a password change is incorrect, then the system shall return a 400 Bad Request response and shall not update the credential.

**REQ-ME-05**
When an authenticated user confirms account self-deletion by providing their current password, the system shall deactivate the account, retain all associated data, and immediately invalidate the user's session tokens.

**REQ-ME-06**
The system shall not allow a self-deleted account to be reactivated without admin intervention.
