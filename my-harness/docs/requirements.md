# 错题本 — 需求文档

**版本**: 1.1
**日期**: 2026-06-24
**描述方式**: EARS（Easy Approach to Requirements Syntax）

---

## EARS 语法说明

| 模式 | 模板 | 适用场景 |
|------|------|---------|
| Ubiquitous（无处不在） | The \<system> shall \<action> | 无条件必须成立的约束 |
| Event-driven（事件驱动） | When \<trigger>, the \<system> shall \<action> | 由事件触发的行为 |
| State-driven（状态驱动） | While \<state>, the \<system> shall \<action> | 持续状态下的行为 |
| Conditional（条件型） | Where \<feature is included>, the \<system> shall \<action> | 可选特性的前提条件 |
| Optional feature（可选功能） | Where \<feature is included>, the \<system> shall \<action> | 同上 |
| Unwanted behavior（非预期行为） | If \<unwanted condition>, then the \<system> shall \<response> | 异常和错误处理 |

---

## 一、用户与认证

### REQ-AUTH-01
The 错题本 system shall require every user to authenticate before accessing any feature.

### REQ-AUTH-02
When a user submits valid credentials, the system shall issue a JWT access token and a refresh token.

### REQ-AUTH-03
When an access token expires, the system shall allow the user to obtain a new access token using a valid refresh token without re-entering credentials.

### REQ-AUTH-04
If a user submits invalid credentials, then the system shall return a 401 Unauthorized response and shall not disclose whether the email or the password is incorrect.

### REQ-AUTH-05
The system shall support three roles: **student**, **teacher**, and **admin**. Each role shall have strictly scoped permissions.

### REQ-AUTH-06
While a user is authenticated, the system shall enforce that all data operations are scoped to that user's own records unless the user holds a teacher or admin role.

---

## 二、错题上传与识别

### REQ-UPLOAD-01
When a student uploads an image file, the system shall validate that the file is JPEG or PNG format and between 1 KB and 10 MB in size before invoking the recognition API.

### REQ-UPLOAD-02
If an uploaded image fails format or size validation, then the system shall return a 400 Bad Request response with a human-readable error message describing the specific violation.

### REQ-UPLOAD-03
When a valid image is uploaded, the system shall invoke the third-party recognition API to extract the question text, subject, and topic tags.

### REQ-UPLOAD-04
When the recognition API returns a confidence score greater than or equal to 0.85, the system shall automatically mark the question as **approved** and store it in the question library.

### REQ-UPLOAD-05
When the recognition API returns a confidence score between 0.50 (inclusive) and 0.85 (exclusive), the system shall store the question with status **pending_review** and place it in the manual review queue.

### REQ-UPLOAD-06
When the recognition API returns a confidence score below 0.50, the system shall reject the upload, return a 422 Unprocessable Entity response, and shall not store the question.

### REQ-UPLOAD-07
If the recognition API is unavailable or returns a non-200 status, then the system shall return an explicit error to the caller and shall not silently substitute default question content.

### REQ-UPLOAD-08
When an image is successfully uploaded, the system shall store the original image file using a server-generated UUID as the filename and shall not incorporate any user-supplied filename into the storage path.

### REQ-UPLOAD-09
The system shall support uploading questions across multiple subjects. The subject shall be detected by the recognition API and stored with each question.

### REQ-UPLOAD-10
When a batch of images is uploaded, the system shall process each image independently and shall report individual success or failure per image without failing the entire batch.

---

## 三、标签管理

### REQ-TAG-01
When a question is recognized, the system shall extract candidate topic tags from the recognition API response and store them as **suggested** tags for that question.

### REQ-TAG-02
When the recognition API returns topic tags, the system shall store all returned tags with **suggested** status regardless of confidence score, and present them to the student for confirmation.

### REQ-TAG-03
When a student views a question with **suggested** tags, the system shall display the suggested tags and allow the student to accept, reject, or modify each tag individually.

### REQ-TAG-04
When a student accepts a suggested tag, the system shall promote that tag to **confirmed** status for that question.

### REQ-TAG-05
When a student rejects a suggested tag, the system shall remove that tag from the question and shall not re-suggest it automatically.

### REQ-TAG-06
The system shall allow a student to manually add free-text tags to any of their own questions at any time.

### REQ-TAG-07
The system shall allow a student to delete any manually added or confirmed tag from their own questions.

### REQ-TAG-08
The system shall include a `has_confirmed_tags` boolean field in the question detail response, set to `false` when the question has no confirmed tags, so that clients can prompt the student to complete tag review.

### REQ-TAG-09
The system shall allow a student to classify a question into one of the following categories: **multiple_choice**, **fill_blank**, **essay**, **true_false**, **calculation**, or **unknown**.

### REQ-TAG-10
When the category returned by the recognition API is **unknown**, the system shall include a prompt indicator in the question response suggesting the student manually select the correct category; the system shall not prevent an unknown-category question from being added to a review paper.

---

## 四、人工审核

### REQ-REVIEW-01
While a question has status **pending_review**, the system shall exclude it from the student's question library and from any review paper generation.

### REQ-REVIEW-02
The system shall provide a review queue interface accessible only to users with the **teacher** or **admin** role.

### REQ-REVIEW-03
When a teacher or admin approves a pending question, the system shall update its status to **approved**, optionally update its category and tags, and record the reviewer's user ID and the review timestamp.

### REQ-REVIEW-04
When a teacher or admin rejects a pending question, the system shall update its status to **rejected**, record the reviewer's user ID, review timestamp, and an optional rejection note.

### REQ-REVIEW-05
If a reviewer attempts to review a question that is not in **pending_review** status, then the system shall return a 409 Conflict response and shall not modify the question.

### REQ-REVIEW-06
The system shall notify the student when one of their questions transitions from **pending_review** to **approved** or **rejected**.

---

## 五、错题搜索

### REQ-SEARCH-01
The system shall allow a student to search their own question library by one or more of the following criteria: subject, topic tag, category, keyword in question text, date range, and review status.

### REQ-SEARCH-02
When a student submits a search query, the system shall return matching questions ordered by creation date descending, with pagination support (default page size: 20, maximum: 100).

### REQ-SEARCH-03
When a search query matches no questions, the system shall return an empty list with count 0 and shall not return an error.

### REQ-SEARCH-04
The system shall support full-text keyword search within the **raw_text** field of questions.

### REQ-SEARCH-05
While a student filters by subject, the system shall only return questions whose subject exactly matches the filter value.

### REQ-SEARCH-06
The system shall allow combining multiple filters in a single search request; all filters shall be applied with AND semantics.

---

## 六、复习试卷编排

### REQ-PAPER-01
The system shall allow a student to create a named review paper by selecting questions from their approved question library.

### REQ-PAPER-02
When a student exports a review paper, the system shall require the paper to contain at least one question. Creating or saving a draft paper with no questions shall be permitted.

### REQ-PAPER-03
The system shall allow a student to add, remove, and reorder questions within a draft review paper before exporting.

### REQ-PAPER-04
The system shall allow a student to filter and search their question library within the paper composition interface, using the same search criteria defined in REQ-SEARCH-01.

### REQ-PAPER-05
When a student exports a review paper, the system shall generate a PDF document containing the selected questions in the chosen order, with each question's subject and category label.

### REQ-PAPER-06
When a PDF is generated, the system shall return the download path to the student in the same response; PDF generation is synchronous.

### REQ-PAPER-07
If a question included in a paper export has status other than **approved**, then the system shall exclude that question from the PDF and notify the student of the exclusion.

### REQ-PAPER-08
The system shall allow a student to save a paper composition as a **draft** and resume editing it in a later session.

### REQ-PAPER-09
The system shall allow a student to duplicate an existing review paper as a starting point for a new paper.

---

## 七、数据隔离与安全

### REQ-SEC-01
The system shall enforce that every database query that reads or writes question, error record, or review data includes a user_id equality condition sourced from the authenticated session token, not from client-supplied input.

### REQ-SEC-02
If a student attempts to access, modify, or delete a resource owned by a different user, then the system shall return a 403 Forbidden response and shall not reveal whether the resource exists.

### REQ-SEC-03
The system shall store all credentials, API keys, and database connection strings in environment variables or AWS Secrets Manager and shall not hard-code them in source code.

### REQ-SEC-04
The system shall reject any request whose JWT is expired, malformed, or signed with an unrecognized key with a 401 Unauthorized response.

---

## 八、错误处理通用约束

### REQ-ERR-01
The system shall return error responses in the format `{"error": "<human-readable message>"}` with a semantically correct HTTP status code (400 / 401 / 403 / 404 / 409 / 422 / 500).

### REQ-ERR-02
The system shall never return a 200 OK response body that contains an error condition.

### REQ-ERR-03
If an internal error occurs that is not caused by client input, then the system shall log the full error details server-side and return only a generic 500 Internal Server Error message to the client.

### REQ-ERR-04
The system shall never silently swallow errors or substitute default values for failed external API calls.

---

## 九、复习调度

### REQ-SCHED-01
When a student submits a review result for a question, the system shall record the result (pass or fail) and compute the next review date using the Ebbinghaus interval: the initial interval is 1 day; on pass, the current interval is doubled; on fail, the interval is reset to 1 day.

### REQ-SCHED-02
When the next review date is computed, the system shall persist the schedule to DynamoDB with a TTL of 30 days beyond the next review date.

### REQ-SCHED-03
The system shall allow a student to retrieve the list of questions scheduled for review on or before the current date.

### REQ-SCHED-04
If a student submits a review result with a value other than **pass** or **fail**, then the system shall return a 400 Bad Request response and shall not update the schedule.

### REQ-SCHED-05
If a student attempts to retrieve or submit a review result for a question owned by a different user, then the system shall return a 403 Forbidden response.

---

## 十、AI 推荐

### REQ-REC-01
The system shall allow a student to request an AI-generated recommendation of up to 5 questions to review next, ranked by predicted review value.

### REQ-REC-02
When generating recommendations, the system shall pass the student's error record summary (question IDs, wrong counts, last wrong dates) to the Bedrock model **claude-sonnet-4-6** and shall return each recommendation with a `question_id`, `reason`, and `confidence` field.

### REQ-REC-03
If the Bedrock response is missing `question_id` or `reason` fields, then the system shall return a 502 Bad Gateway error and shall not return a partial recommendation list.

---

## 十一、错题记录

### REQ-ERR-REC-01
The system shall maintain an error record per (user, question) pair, tracking the cumulative wrong count and the date of the most recent error.

### REQ-ERR-REC-02
The system shall allow a student to retrieve their own error records; all queries shall be scoped to the authenticated user's ID.

---

## 十二、通知

### REQ-NOTIF-01
The system shall allow a student to retrieve their notification list, with support for filtering to unread notifications only and pagination (default page size: 20, maximum: 100).

### REQ-NOTIF-02
When a student marks a notification as read, the system shall update that notification's read status and shall return 204 No Content.

### REQ-NOTIF-03
The system shall allow a student to mark all of their notifications as read in a single operation.

---

## 十三、账号与用户管理

### REQ-ACCT-01
The system shall allow an admin to create a user account by specifying an email address and role; the system shall generate a random initial password and return it in the creation response, displayed only once.

### REQ-ACCT-02
The system shall allow an admin to import user accounts in bulk by uploading a CSV file containing email and role columns; each row shall be processed independently and the response shall list succeeded and failed rows without aborting the entire batch. A single import request shall not exceed 200 rows.

### REQ-ACCT-03
When an admin deactivates a user account, the system shall immediately revoke the user's ability to authenticate and shall retain all of that user's question and review data.

### REQ-ACCT-04
When an admin reactivates a deactivated account, the system shall restore the user's ability to authenticate and all previously retained data shall remain accessible.

### REQ-ACCT-05
The system shall allow an admin to change a user's role to student, teacher, or admin.

### REQ-ACCT-06
The system shall allow an admin to list all user accounts with filtering by role and status, with pagination (default page size: 20, maximum: 100).

### REQ-ACCT-07
The system shall allow an admin to view the question list of any user, including deactivated users, in read-only mode.

### REQ-ACCT-08
If a non-admin user attempts to access any `/api/admin/*` endpoint, then the system shall return a 403 Forbidden response.

---

## 十四、用户自管理

### REQ-ME-01
The system shall allow an authenticated user to retrieve their own profile (id, email, nickname, role, created_at).

### REQ-ME-02
The system shall allow an authenticated user to update their nickname (1–50 characters).

### REQ-ME-03
When an authenticated user submits a password change request with their current password and a new password (8–72 characters, at least one letter and one digit), the system shall update the credential via Cognito and return 204 No Content.

### REQ-ME-04
If the current password provided during a password change is incorrect, then the system shall return a 400 Bad Request response and shall not update the credential.

### REQ-ME-05
When an authenticated user confirms account self-deletion by providing their current password, the system shall deactivate the account, retain all associated data, and immediately invalidate the user's session tokens.

### REQ-ME-06
The system shall not allow a self-deleted account to be reactivated without admin intervention.

---

## 十五、学习统计看板

### REQ-STATS-01
The system shall allow a student to retrieve their learning statistics for a custom date range (maximum span: 365 days), including daily new question counts, per-subject mastery rates, and today's pending review count.

### REQ-STATS-02
The system shall compute per-subject mastery rate using an interval-weighted formula: each question's weight is `log2(current_interval_days + 1)`; questions with no review history contribute weight 0; the result is normalized to the range [0.0, 1.0].

### REQ-STATS-03
The system shall return today's pending review count as a single integer derived from the review schedule, without returning the full question list.

### REQ-STATS-04
If the requested date range exceeds 365 days, then the system shall truncate to 365 days from `date_from` and include a `truncated: true` flag in the response.

### REQ-STATS-05
The system shall allow a teacher or admin to retrieve a summary list of all students' statistics for a given date range, including question count, average mastery rate, today's pending review count, and last active date per student.

### REQ-STATS-06
The system shall allow a teacher or admin to retrieve the full statistics detail of any individual student, using the same structure as the student's own statistics response.

### REQ-STATS-07
If a student attempts to access `/api/stats/class/*`, then the system shall return a 403 Forbidden response.

### REQ-STATS-08
If a teacher or admin requests statistics for a user who does not exist or is not a student, then the system shall return a 404 Not Found response.

---

## 十六、班级管理

### REQ-CLASS-01
The system shall allow a teacher to create a named class; the system shall generate a unique 6-character alphanumeric invite code for the class.

### REQ-CLASS-02
The system shall allow a student to join a class by submitting a valid invite code; a student may belong to multiple classes simultaneously.

### REQ-CLASS-03
When a teacher resets a class invite code, the system shall generate a new unique code and immediately invalidate the previous code.

### REQ-CLASS-04
The system shall allow a teacher to view the member list of their own class and remove any student from it.

### REQ-CLASS-05
The system shall allow a student to leave a class voluntarily; leaving a class shall not delete the student's submitted task results.

### REQ-CLASS-06
If a user attempts to join a class with an invalid or expired invite code, then the system shall return a 404 Not Found response.

### REQ-CLASS-07
If a student attempts to create a class, delete a member, or reset an invite code, then the system shall return a 403 Forbidden response.

---

## 十七、班级任务

### REQ-TASK-01
The system shall allow a teacher to assign a review task to a class by selecting one of their own papers, with an optional due date.

### REQ-TASK-02
The system shall allow a student to view the task list for each of their classes, including each task's title, due date, and their own completion status.

### REQ-TASK-03
The system shall allow a student to submit pass or fail results for each question in a task; each (task, student, question) combination shall only be submitted once.

### REQ-TASK-04
When a student submits a task question result, the system shall update the student's personal Ebbinghaus review schedule for that question, following the same rules as REQ-SCHED-01.

### REQ-TASK-05
If a student attempts to submit results for a task that is closed or whose due date has passed, then the system shall return a 403 Forbidden response.

### REQ-TASK-06
If a student attempts to submit results for the same question in the same task more than once, then the system shall return a 409 Conflict response.

### REQ-TASK-07
The system shall allow a teacher to view per-student completion progress for any task in their class, including submitted count, total question count, and pass count per student.

### REQ-TASK-08
The system shall allow a teacher to update the due date of an active task or close it early; a closed task shall not accept further student submissions and shall not be reopened.

---

## 十八、非功能性需求

### REQ-NFR-01
The system shall respond to all API requests (excluding PDF generation and recognition API calls) within 500 ms at the 95th percentile under normal load.

### REQ-NFR-02
The system shall use Go 1.25 for all backend services.

### REQ-NFR-03
The system shall be deployed on AWS ECS Fargate with MySQL (RDS) as the primary relational store and DynamoDB for review scheduling.

### REQ-NFR-04
The system shall use structured JSON logging (slog) for all server-side log output, including user_id on every log entry.

### REQ-NFR-05
Where the PDF export feature is included, the system shall generate a downloadable PDF within 10 seconds for papers containing up to 100 questions.
