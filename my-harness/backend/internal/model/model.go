package model

import "time"

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

type User struct {
	ID            string     `json:"id"             db:"id"`
	CognitoSub    string     `json:"cognito_sub"    db:"cognito_sub"`
	Email         string     `json:"email"          db:"email"`
	Nickname      string     `json:"nickname"       db:"nickname"`
	Role          Role       `json:"role"           db:"role"`
	Status        UserStatus `json:"status"         db:"status"`
	DeactivatedAt *time.Time `json:"deactivated_at" db:"deactivated_at"`
	CreatedAt     time.Time  `json:"created_at"     db:"created_at"`
}

// QuestionStatus 审核状态
type QuestionStatus string

const (
	QuestionStatusPendingReview QuestionStatus = "pending_review"
	QuestionStatusApproved      QuestionStatus = "approved"
	QuestionStatusRejected      QuestionStatus = "rejected"
)

// QuestionCategory 题目类型
type QuestionCategory string

const (
	CategoryMultipleChoice QuestionCategory = "multiple_choice"
	CategoryFillBlank      QuestionCategory = "fill_blank"
	CategoryEssay          QuestionCategory = "essay"
	CategoryTrueFalse      QuestionCategory = "true_false"
	CategoryCalculation    QuestionCategory = "calculation"
	CategoryUnknown        QuestionCategory = "unknown"
)

type Question struct {
	ID         string           `json:"id"          db:"id"`
	UserID     string           `json:"user_id"     db:"user_id"`
	ImagePath  string           `json:"image_path"  db:"image_path"`
	RawText    string           `json:"raw_text"    db:"raw_text"`
	Subject    string           `json:"subject"     db:"subject"`
	Source     string           `json:"source"      db:"source"`
	Status     QuestionStatus   `json:"status"      db:"status"`
	Category   QuestionCategory `json:"category"    db:"category"`
	Confidence float64          `json:"confidence"  db:"confidence"`
	ReviewNote string           `json:"review_note" db:"review_note"`
	ReviewedBy string           `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt *time.Time       `json:"reviewed_at" db:"reviewed_at"`
	CreatedAt  time.Time        `json:"created_at"  db:"created_at"`
}

// TagStatus 标签状态
type TagStatus string

const (
	TagStatusSuggested TagStatus = "suggested"
	TagStatusConfirmed TagStatus = "confirmed"
)

// Tag 题目标签，支持 suggested（待确认）和 confirmed（已确认）两种状态
type Tag struct {
	ID         string    `json:"id"          db:"id"`
	QuestionID string    `json:"question_id" db:"question_id"`
	UserID     string    `json:"user_id"     db:"user_id"`
	Name       string    `json:"name"        db:"name"`
	Status     TagStatus `json:"status"      db:"status"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

// PaperStatus 试卷状态
type PaperStatus string

const (
	PaperStatusDraft     PaperStatus = "draft"
	PaperStatusPublished PaperStatus = "published"
)

// Paper 复习试卷
type Paper struct {
	ID        string      `json:"id"         db:"id"`
	UserID    string      `json:"user_id"    db:"user_id"`
	Title     string      `json:"title"      db:"title"`
	Status    PaperStatus `json:"status"     db:"status"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

// PaperQuestion 试卷中的题目，position 决定顺序
type PaperQuestion struct {
	ID         string    `json:"id"          db:"id"`
	PaperID    string    `json:"paper_id"    db:"paper_id"`
	QuestionID string    `json:"question_id" db:"question_id"`
	Position   int       `json:"position"    db:"position"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	// 关联题目详情（查询时填充，不存数据库）
	Question *Question `json:"question,omitempty" db:"-"`
}

// NotificationType 通知类型
type NotificationType string

const (
	NotifTypeQuestionApproved NotificationType = "question_approved"
	NotifTypeQuestionRejected NotificationType = "question_rejected"
)

// Notification 站内通知
type Notification struct {
	ID        string           `json:"id"         db:"id"`
	UserID    string           `json:"user_id"    db:"user_id"`
	Type      NotificationType `json:"type"       db:"type"`
	Title     string           `json:"title"      db:"title"`
	Body      string           `json:"body"       db:"body"`
	RefID     string           `json:"ref_id"     db:"ref_id"`
	IsRead    bool             `json:"is_read"    db:"is_read"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}

// SearchParams 多维搜索参数
type SearchParams struct {
	UserID   string
	Subject  string
	Category QuestionCategory
	Status   QuestionStatus
	Tag      string
	Keyword  string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PageSize int
}

type ErrorRecord struct {
	ID          string    `json:"id"           db:"id"`
	UserID      string    `json:"user_id"      db:"user_id"`
	QuestionID  string    `json:"question_id"  db:"question_id"`
	WrongCount  int       `json:"wrong_count"  db:"wrong_count"`
	LastWrongAt time.Time `json:"last_wrong_at" db:"last_wrong_at"`
	CreatedAt   time.Time `json:"created_at"   db:"created_at"`
}

type ReviewRecord struct {
	ID         string    `json:"id"          db:"id"`
	UserID     string    `json:"user_id"     db:"user_id"`
	QuestionID string    `json:"question_id" db:"question_id"`
	ReviewedAt time.Time `json:"reviewed_at" db:"reviewed_at"`
	Result     string    `json:"result"      db:"result"`
}

type ReviewSchedule struct {
	UserID       string    `dynamodbav:"user_id"`
	NextReviewAt time.Time `dynamodbav:"next_review_at"`
	QuestionID   string    `dynamodbav:"question_id"`
	IntervalDays int       `dynamodbav:"interval_days"`
	TTL          int64     `dynamodbav:"ttl"`
}

type ReviewResult string

const (
	ReviewPass ReviewResult = "pass"
	ReviewFail ReviewResult = "fail"
)

type ReviewAction string

const (
	ReviewActionApprove ReviewAction = "approve"
	ReviewActionReject  ReviewAction = "reject"
)

type ClassTaskStatus string

const (
	ClassTaskStatusActive ClassTaskStatus = "active"
	ClassTaskStatusClosed ClassTaskStatus = "closed"
)

type Class struct {
	ID         string    `json:"id"          db:"id"`
	Name       string    `json:"name"        db:"name"`
	TeacherID  string    `json:"teacher_id"  db:"teacher_id"`
	InviteCode string    `json:"invite_code" db:"invite_code"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

type ClassMember struct {
	ID       string    `json:"id"        db:"id"`
	ClassID  string    `json:"class_id"  db:"class_id"`
	UserID   string    `json:"user_id"   db:"user_id"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}

type ClassTask struct {
	ID         string          `json:"id"          db:"id"`
	ClassID    string          `json:"class_id"    db:"class_id"`
	PaperID    string          `json:"paper_id"    db:"paper_id"`
	Title      string          `json:"title"       db:"title"`
	AssignedBy string          `json:"assigned_by" db:"assigned_by"`
	DueAt      *time.Time      `json:"due_at"      db:"due_at"`
	Status     ClassTaskStatus `json:"status"      db:"status"`
	CreatedAt  time.Time       `json:"created_at"  db:"created_at"`
}

type TaskSubmission struct {
	ID          string    `json:"id"           db:"id"`
	TaskID      string    `json:"task_id"      db:"task_id"`
	UserID      string    `json:"user_id"      db:"user_id"`
	QuestionID  string    `json:"question_id"  db:"question_id"`
	Result      string    `json:"result"       db:"result"`
	SubmittedAt time.Time `json:"submitted_at" db:"submitted_at"`
}

type EmbeddingJob struct {
	QuestionID string `dynamodbav:"question_id"`
	SK         string `dynamodbav:"sk"` // fixed value "job"
	UserID     string `dynamodbav:"user_id"`
	Status     string `dynamodbav:"status"` // pending | done | failed
	RetryCount int    `dynamodbav:"retry_count"`
	CreatedAt  string `dynamodbav:"created_at"` // ISO8601
	TTL        int64  `dynamodbav:"ttl"`
}
