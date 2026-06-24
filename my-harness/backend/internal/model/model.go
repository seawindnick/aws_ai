package model

import "time"

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID         string    `db:"id"`
	CognitoSub string    `db:"cognito_sub"`
	Email      string    `db:"email"`
	Role       Role      `db:"role"`
	SchoolName string    `db:"school_name"`
	CreatedAt  time.Time `db:"created_at"`
}

// QuestionStatus 表示题目的审核状态。
// pending_review: 识别置信度不足，等待人工审核。
// approved: 自动或人工审核通过，进入题库。
// rejected: 人工审核拒绝，不进入题库。
type QuestionStatus string

const (
	QuestionStatusPendingReview QuestionStatus = "pending_review"
	QuestionStatusApproved      QuestionStatus = "approved"
	QuestionStatusRejected      QuestionStatus = "rejected"
)

// QuestionCategory 题目类型，由识别 API 返回或人工审核时修正。
type QuestionCategory string

const (
	CategoryMultipleChoice QuestionCategory = "multiple_choice" // 选择题
	CategoryFillBlank      QuestionCategory = "fill_blank"      // 填空题
	CategoryEssay          QuestionCategory = "essay"           // 解答题
	CategoryTrueFalse      QuestionCategory = "true_false"      // 判断题
	CategoryCalculation    QuestionCategory = "calculation"     // 计算题
	CategoryUnknown        QuestionCategory = "unknown"         // 无法识别类型
)

type Question struct {
	ID         string           `db:"id"`
	UserID     string           `db:"user_id"`
	ImagePath  string           `db:"image_path"`
	RawText    string           `db:"raw_text"`
	Subject    string           `db:"subject"`
	TopicTags  []string         `db:"topic_tags"`
	Source     string           `db:"source"`
	Status     QuestionStatus   `db:"status"`
	Category   QuestionCategory `db:"category"`
	Confidence float64          `db:"confidence"`
	ReviewNote string           `db:"review_note"`
	ReviewedBy string           `db:"reviewed_by"`
	ReviewedAt *time.Time       `db:"reviewed_at"`
	CreatedAt  time.Time        `db:"created_at"`
}

type ErrorRecord struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	QuestionID  string    `db:"question_id"`
	WrongCount  int       `db:"wrong_count"`
	LastWrongAt time.Time `db:"last_wrong_at"`
	CreatedAt   time.Time `db:"created_at"`
}

type ReviewRecord struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	QuestionID string    `db:"question_id"`
	ReviewedAt time.Time `db:"reviewed_at"`
	Result     string    `db:"result"` // "pass" | "fail"
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

// ReviewAction 人工审核操作。
type ReviewAction string

const (
	ReviewActionApprove ReviewAction = "approve"
	ReviewActionReject  ReviewAction = "reject"
)
