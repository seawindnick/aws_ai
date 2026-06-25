package service

import (
	"context"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type ErrorRecordService struct {
	repo *repository.ErrorRecordRepo
}

func NewErrorRecordService(repo *repository.ErrorRecordRepo) *ErrorRecordService {
	return &ErrorRecordService{repo: repo}
}

func (s *ErrorRecordService) List(ctx context.Context, userID string, page, pageSize int) ([]*model.ErrorRecord, error) {
	records, err := s.repo.ListByUser(ctx, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list error records: %w", err)
	}
	return records, nil
}
