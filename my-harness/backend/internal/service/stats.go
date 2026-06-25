package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type StatsService struct {
	questionRepo *repository.QuestionRepo
	reviewRepo   *repository.ReviewRepo
	userRepo     *repository.UserRepo
}

func NewStatsService(
	questionRepo *repository.QuestionRepo,
	reviewRepo *repository.ReviewRepo,
	userRepo *repository.UserRepo,
) *StatsService {
	return &StatsService{questionRepo: questionRepo, reviewRepo: reviewRepo, userRepo: userRepo}
}

type StudentStats struct {
	DailyNewQuestions []DailyCount     `json:"daily_new_questions"`
	SubjectMastery    []SubjectMastery `json:"subject_mastery"`
	TodayPendingCount int              `json:"today_pending_count"`
	Truncated         bool             `json:"truncated,omitempty"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type SubjectMastery struct {
	Subject    string  `json:"subject"`
	MasteryRate float64 `json:"mastery_rate"`
}

type ClassStudentSummary struct {
	StudentID          string  `json:"student_id"`
	Email              string  `json:"email"`
	Nickname           string  `json:"nickname"`
	QuestionCount      int     `json:"question_count"`
	AvgMasteryRate     float64 `json:"avg_mastery_rate"`
	TodayPendingCount  int     `json:"today_pending_count"`
	LastActiveDate     string  `json:"last_active_date"`
}

func (s *StatsService) GetStudentStats(ctx context.Context, userID string, dateFrom, dateTo time.Time) (*StudentStats, error) {
	truncated := false
	if dateTo.Sub(dateFrom) > 365*24*time.Hour {
		dateTo = dateFrom.Add(365 * 24 * time.Hour)
		truncated = true
	}

	rawDaily, err := s.questionRepo.DailyNewCounts(ctx, userID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("daily counts: %w", err)
	}
	daily := make([]DailyCount, 0, len(rawDaily))
	for _, d := range rawDaily {
		daily = append(daily, DailyCount{Date: d.Date, Count: d.Count})
	}

	mastery, err := s.computeMastery(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("compute mastery: %w", err)
	}

	todaySchedules, err := s.reviewRepo.ListTodaySchedule(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("today schedule: %w", err)
	}

	return &StudentStats{
		DailyNewQuestions: daily,
		SubjectMastery:    mastery,
		TodayPendingCount: len(todaySchedules),
		Truncated:         truncated,
	}, nil
}

func (s *StatsService) GetClassStats(ctx context.Context, dateFrom, dateTo time.Time) ([]*ClassStudentSummary, error) {
	students, err := s.userRepo.List(ctx, string(model.RoleStudent), string(model.UserStatusActive), 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}

	var result []*ClassStudentSummary
	for _, st := range students {
		stats, err := s.GetStudentStats(ctx, st.ID, dateFrom, dateTo)
		if err != nil {
			continue
		}
		total := 0
		for _, d := range stats.DailyNewQuestions {
			total += d.Count
		}
		avgMastery := 0.0
		if len(stats.SubjectMastery) > 0 {
			for _, m := range stats.SubjectMastery {
				avgMastery += m.MasteryRate
			}
			avgMastery /= float64(len(stats.SubjectMastery))
		}
		result = append(result, &ClassStudentSummary{
			StudentID:         st.ID,
			Email:             st.Email,
			Nickname:          st.Nickname,
			QuestionCount:     total,
			AvgMasteryRate:    math.Round(avgMastery*100) / 100,
			TodayPendingCount: stats.TodayPendingCount,
		})
	}
	return result, nil
}

func (s *StatsService) GetStudentStatsByID(ctx context.Context, studentID string, dateFrom, dateTo time.Time) (*StudentStats, error) {
	u, err := s.userRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("student not found")
	}
	if u.Role != model.RoleStudent {
		return nil, fmt.Errorf("user is not a student")
	}
	return s.GetStudentStats(ctx, studentID, dateFrom, dateTo)
}

const maxIntervalDays = 64

func (s *StatsService) computeMastery(ctx context.Context, userID string) ([]SubjectMastery, error) {
	questions, err := s.questionRepo.ListApprovedByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, nil
	}

	// Collect all question IDs and batch-fetch schedules to avoid N+1 DynamoDB calls.
	ids := make([]string, 0, len(questions))
	for _, q := range questions {
		ids = append(ids, q.ID)
	}
	intervalByID, err := s.reviewRepo.BatchGetSchedules(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("batch get schedules: %w", err)
	}

	type subjectAgg struct {
		totalWeight float64
		count       int
	}
	bySubject := map[string]*subjectAgg{}
	maxWeight := math.Log2(float64(maxIntervalDays) + 1)

	for _, q := range questions {
		agg, ok := bySubject[q.Subject]
		if !ok {
			agg = &subjectAgg{}
			bySubject[q.Subject] = agg
		}
		agg.count++
		if interval, found := intervalByID[q.ID]; found && interval > 0 {
			agg.totalWeight += math.Log2(float64(interval) + 1)
		}
	}

	var result []SubjectMastery
	for subj, agg := range bySubject {
		rate := 0.0
		if agg.count > 0 && maxWeight > 0 {
			rate = agg.totalWeight / (float64(agg.count) * maxWeight)
			if rate > 1.0 {
				rate = 1.0
			}
		}
		result = append(result, SubjectMastery{
			Subject:     subj,
			MasteryRate: math.Round(rate*100) / 100,
		})
	}
	return result, nil
}
