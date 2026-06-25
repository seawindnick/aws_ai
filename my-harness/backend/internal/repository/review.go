package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/workshop/wrong-question/internal/model"
)

type ReviewRepo struct {
	db        *sql.DB
	ddb       *dynamodb.Client
	tableName string
}

func NewReviewRepo(db *sql.DB, ddb *dynamodb.Client, tableName string) *ReviewRepo {
	return &ReviewRepo{db: db, ddb: ddb, tableName: tableName}
}

func (r *ReviewRepo) SaveRecord(ctx context.Context, rec *model.ReviewRecord) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO review_records (id, user_id, question_id, reviewed_at, result)
		 VALUES (?, ?, ?, ?, ?)`,
		rec.ID, rec.UserID, rec.QuestionID, rec.ReviewedAt, rec.Result,
	)
	if err != nil {
		return fmt.Errorf("insert review record: %w", err)
	}
	return nil
}

func (r *ReviewRepo) GetSchedule(ctx context.Context, userID, questionID string) (*model.ReviewSchedule, error) {
	out, err := r.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"user_id":     &types.AttributeValueMemberS{Value: userID},
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}
	s := &model.ReviewSchedule{}
	if err := attributevalue.UnmarshalMap(out.Item, s); err != nil {
		return nil, fmt.Errorf("unmarshal schedule: %w", err)
	}
	return s, nil
}

// BatchGetSchedules returns a map[questionID]IntervalDays for the given question IDs.
// Uses DynamoDB BatchGetItem (max 100 per call) to avoid N+1 per-question lookups.
func (r *ReviewRepo) BatchGetSchedules(ctx context.Context, userID string, questionIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(questionIDs))
	if len(questionIDs) == 0 {
		return result, nil
	}

	// DynamoDB BatchGetItem limit: 100 keys per call
	const batchSize = 100
	for start := 0; start < len(questionIDs); start += batchSize {
		end := start + batchSize
		if end > len(questionIDs) {
			end = len(questionIDs)
		}
		batch := questionIDs[start:end]

		keys := make([]map[string]types.AttributeValue, 0, len(batch))
		for _, qid := range batch {
			keys = append(keys, map[string]types.AttributeValue{
				"user_id":     &types.AttributeValueMemberS{Value: userID},
				"question_id": &types.AttributeValueMemberS{Value: qid},
			})
		}

		out, err := r.ddb.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				r.tableName: {Keys: keys},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("batch get schedules: %w", err)
		}

		for _, item := range out.Responses[r.tableName] {
			s := &model.ReviewSchedule{}
			if err := attributevalue.UnmarshalMap(item, s); err != nil {
				continue
			}
			result[s.QuestionID] = s.IntervalDays
		}
	}
	return result, nil
}

func (r *ReviewRepo) SaveSchedule(ctx context.Context, s *model.ReviewSchedule) error {
	item, err := attributevalue.MarshalMap(s)
	if err != nil {
		return fmt.Errorf("marshal schedule: %w", err)
	}
	_, err = r.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put schedule: %w", err)
	}
	return nil
}

func (r *ReviewRepo) DeleteSchedule(ctx context.Context, userID, questionID string) error {
	_, err := r.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"user_id":     &types.AttributeValueMemberS{Value: userID},
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
	})
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

func (r *ReviewRepo) ListTodaySchedule(ctx context.Context, userID string) ([]*model.ReviewSchedule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	out, err := r.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              aws.String("user_date_index"),
		KeyConditionExpression: aws.String("user_id = :uid AND next_review_at <= :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
			":now": &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query today schedule: %w", err)
	}

	var result []*model.ReviewSchedule
	for _, item := range out.Items {
		s := &model.ReviewSchedule{}
		if err := attributevalue.UnmarshalMap(item, s); err != nil {
			return nil, fmt.Errorf("unmarshal schedule: %w", err)
		}
		result = append(result, s)
	}
	return result, nil
}
