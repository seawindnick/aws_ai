package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/workshop/wrong-question/internal/model"
)

type EmbeddingJobService struct {
	ddb       *dynamodb.Client
	tableName string
}

func NewEmbeddingJobService(ddb *dynamodb.Client, tableName string) *EmbeddingJobService {
	return &EmbeddingJobService{ddb: ddb, tableName: tableName}
}

func (s *EmbeddingJobService) Enqueue(ctx context.Context, questionID, userID string) error {
	now := time.Now().UTC()
	job := &model.EmbeddingJob{
		QuestionID: questionID,
		SK:         "job",
		UserID:     userID,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  now.Format(time.RFC3339),
		TTL:        now.Add(7 * 24 * time.Hour).Unix(),
	}
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		return fmt.Errorf("marshal embedding job: %w", err)
	}
	_, err = s.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put embedding job: %w", err)
	}
	return nil
}

func (s *EmbeddingJobService) Delete(ctx context.Context, questionID string) error {
	item, _ := attributevalue.MarshalMap(map[string]string{
		"question_id": questionID,
		"sk":          "job",
	})
	_, err := s.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key:       item,
	})
	return err
}
