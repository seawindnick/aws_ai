package model

import "time"

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID          string    `db:"id"`
	CognitoSub  string    `db:"cognito_sub"`
	Email       string    `db:"email"`
	Role        Role      `db:"role"`
	SchoolName  string    `db:"school_name"`
	CreatedAt   time.Time `db:"created_at"`
}

type Question struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	ImagePath  string    `db:"image_path"`
	RawText    string    `db:"raw_text"`
	Subject    string    `db:"subject"`
	TopicTags  []string  `db:"topic_tags"`
	Source     string    `db:"source"`
	CreatedAt  time.Time `db:"created_at"`
}

type ErrorRecord struct {
	ID           string    `db:"id"`
	UserID       string    `db:"user_id"`
	QuestionID   string    `db:"question_id"`
	WrongCount   int       `db:"wrong_count"`
	LastWrongAt  time.Time `db:"last_wrong_at"`
	CreatedAt    time.Time `db:"created_at"`
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
