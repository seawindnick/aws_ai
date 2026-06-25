package service

import (
	"context"
	"fmt"

	"github.com/workshop/wrong-question/internal/repository"
)

type RecommendService struct {
	bedrockSvc *BedrockService
	errorRepo  *repository.ErrorRecordRepo
}

func NewRecommendService(bedrockSvc *BedrockService, errorRepo *repository.ErrorRecordRepo) *RecommendService {
	return &RecommendService{bedrockSvc: bedrockSvc, errorRepo: errorRepo}
}

func (s *RecommendService) Recommend(ctx context.Context, userID string) ([]*RecommendItem, error) {
	errors, err := s.errorRepo.SummarizeByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get error records: %w", err)
	}

	summary := fmt.Sprintf("User has %d error records.", len(errors))
	for _, e := range errors {
		summary += fmt.Sprintf("\n- question_id: %s, wrong_count: %d, last_wrong: %s",
			e.QuestionID, e.WrongCount, e.LastWrongAt.Format("2006-01-02"))
	}

	return s.bedrockSvc.Recommend(ctx, summary)
}
