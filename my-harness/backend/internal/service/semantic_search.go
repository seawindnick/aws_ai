package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type SemanticResult struct {
	Question *model.Question `json:"question"`
	Score    float64         `json:"score"`
}

type SemanticSearchService struct {
	embedSvc     *EmbeddingService
	vectorRepo   *repository.S3VectorsRepo
	questionRepo *repository.QuestionRepo
}

func NewSemanticSearchService(
	embedSvc *EmbeddingService,
	vectorRepo *repository.S3VectorsRepo,
	questionRepo *repository.QuestionRepo,
) *SemanticSearchService {
	return &SemanticSearchService{
		embedSvc:     embedSvc,
		vectorRepo:   vectorRepo,
		questionRepo: questionRepo,
	}
}

func (s *SemanticSearchService) Search(ctx context.Context, userID, query, subject string) ([]*SemanticResult, error) {
	if len(query) == 0 || len(query) > 500 {
		return nil, fmt.Errorf("query must be 1–500 characters")
	}

	vector, err := s.embedSvc.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	hits, err := s.vectorRepo.Query(ctx, userID, subject, vector, 5)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	if len(hits) == 0 {
		return []*SemanticResult{}, nil
	}

	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.QuestionID)
	}

	questions, err := s.questionRepo.GetByIDs(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch questions: %w", err)
	}

	qMap := make(map[string]*model.Question, len(questions))
	for _, q := range questions {
		qMap[q.ID] = q
	}

	var results []*SemanticResult
	for _, h := range hits {
		q, ok := qMap[h.QuestionID]
		if !ok {
			continue // question deleted, skip silently
		}
		results = append(results, &SemanticResult{Question: q, Score: h.Score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results, nil
}
