package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	s3vdoc "github.com/aws/aws-sdk-go-v2/service/s3vectors/document"
	s3vtypes "github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
)

var (
	ddbClient      *dynamodb.Client
	bedrockClient  *bedrockruntime.Client
	s3vClient      *s3vectors.Client
	questionsTable string
	vectorBucket   string
	embeddingModel string
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		slog.Error("load aws config", "error", err)
		os.Exit(1)
	}

	ddbClient = dynamodb.NewFromConfig(cfg)
	bedrockClient = bedrockruntime.NewFromConfig(cfg)
	s3vClient = s3vectors.NewFromConfig(cfg)

	questionsTable = mustEnv("DYNAMO_TABLE_QUESTIONS")
	vectorBucket = getEnv("S3_VECTORS_BUCKET", "")
	embeddingModel = getEnv("EMBEDDING_MODEL_ID", "amazon.titan-embed-text-v2:0")
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		// Only process MODIFY events — fired when image-analyzer sets status=done
		if record.EventName != "MODIFY" {
			continue
		}

		// Convert events.DynamoDBAttributeValue map to SDK types for unmarshalling
		av := make(map[string]types.AttributeValue)
		for k, v := range record.Change.NewImage {
			av[k] = convertAttrValue(v)
		}

		var q struct {
			QuestionID string `dynamodbav:"question_id"`
			UserID     string `dynamodbav:"user_id"`
			Subject    string `dynamodbav:"subject"`
			RawText    string `dynamodbav:"raw_text"`
			Status     string `dynamodbav:"status"`
		}
		if err := attributevalue.UnmarshalMap(av, &q); err != nil {
			slog.Error("unmarshal question", "error", err)
			continue
		}

		if q.Status != "done" || q.RawText == "" {
			continue
		}

		if vectorBucket == "" {
			slog.Warn("S3_VECTORS_BUCKET not set, skipping embedding", "question_id", q.QuestionID)
			continue
		}

		if err := embedQuestion(ctx, q.QuestionID, q.UserID, q.Subject, q.RawText); err != nil {
			slog.Error("embed question", "question_id", q.QuestionID, "error", err)
			markEmbedStatus(ctx, q.QuestionID, "embed_failed")
		}
	}
	return nil
}

func embedQuestion(ctx context.Context, questionID, userID, subject, rawText string) error {
	body, _ := json.Marshal(map[string]string{"inputText": rawText})
	out, err := bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(embeddingModel),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("bedrock embed: %w", err)
	}

	var resp struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return fmt.Errorf("decode embedding: %w", err)
	}

	_, err = s3vClient.PutVectors(ctx, &s3vectors.PutVectorsInput{
		VectorBucketName: aws.String(vectorBucket),
		Vectors: []s3vtypes.PutInputVector{
			{
				Key:  aws.String(questionID),
				Data: &s3vtypes.VectorDataMemberFloat32{Value: resp.Embedding},
				Metadata: s3vdoc.NewLazyDocument(map[string]any{
					"user_id": userID,
					"subject": subject,
				}),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("put vector: %w", err)
	}

	slog.Info("embedding stored", "question_id", questionID)
	return nil
}

func markEmbedStatus(ctx context.Context, questionID, status string) {
	_, err := ddbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(questionsTable),
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
		UpdateExpression: aws.String("SET embed_status = :s, updated_at = :t"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: status},
			":t": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		slog.Error("mark embed status", "question_id", questionID, "error", err)
	}
}

func convertAttrValue(v events.DynamoDBAttributeValue) types.AttributeValue {
	switch v.DataType() {
	case events.DataTypeString:
		return &types.AttributeValueMemberS{Value: v.String()}
	case events.DataTypeNumber:
		return &types.AttributeValueMemberN{Value: v.Number()}
	case events.DataTypeBoolean:
		return &types.AttributeValueMemberBOOL{Value: v.Boolean()}
	default:
		return &types.AttributeValueMemberS{Value: ""}
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var " + key + " not set")
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})))
	lambda.Start(handler)
}
