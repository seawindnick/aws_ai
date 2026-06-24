package main

import (
	"context"
	"database/sql"
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
	s3vtypes "github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
	_ "github.com/go-sql-driver/mysql"
)

var (
	ddbClient      *dynamodb.Client
	bedrockClient  *bedrockruntime.Client
	s3vClient      *s3vectors.Client
	db             *sql.DB
	embedJobsTable string
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=True&loc=UTC",
		mustEnv("DB_USER"), mustEnv("DB_PASSWORD"),
		mustEnv("DB_HOST"), getEnv("DB_PORT", "3306"),
		mustEnv("DB_NAME"),
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("open mysql", "error", err)
		os.Exit(1)
	}

	embedJobsTable = mustEnv("DYNAMO_TABLE_EMBED_JOBS")
	vectorBucket = mustEnv("S3_VECTORS_BUCKET")
	embeddingModel = getEnv("EMBEDDING_MODEL_ID", "amazon.titan-embed-text-v2:0")
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		var job struct {
			QuestionID string `dynamodbav:"question_id"`
			UserID     string `dynamodbav:"user_id"`
			Status     string `dynamodbav:"status"`
			RetryCount int    `dynamodbav:"retry_count"`
		}

		// Convert events.DynamoDBAttributeValue map to SDK types
		av := make(map[string]types.AttributeValue)
		for k, v := range record.Change.NewImage {
			av[k] = convertAttrValue(v)
		}
		if err := attributevalue.UnmarshalMap(av, &job); err != nil {
			slog.Error("unmarshal job", "error", err)
			continue
		}

		if job.Status != "pending" {
			continue
		}

		if err := processJob(ctx, job.QuestionID, job.UserID, job.RetryCount); err != nil {
			slog.Error("process embedding job", "question_id", job.QuestionID, "error", err)
			updateJobStatus(ctx, job.QuestionID, "failed", job.RetryCount+1)
		}
	}
	return nil
}

func processJob(ctx context.Context, questionID, userID string, retryCount int) error {
	if retryCount >= 3 {
		slog.Warn("max retries reached", "question_id", questionID)
		return nil
	}

	var rawText, subject string
	err := db.QueryRowContext(ctx,
		`SELECT raw_text, subject FROM questions WHERE id = ?`, questionID,
	).Scan(&rawText, &subject)
	if err != nil {
		return fmt.Errorf("read question: %w", err)
	}

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

	meta, _ := json.Marshal(map[string]string{"user_id": userID, "subject": subject})
	_, err = s3vClient.PutVectors(ctx, &s3vectors.PutVectorsInput{
		VectorBucketName: aws.String(vectorBucket),
		Vectors: []s3vtypes.PutInputVector{
			{
				Key:      aws.String(questionID),
				Data:     &s3vtypes.VectorDataMemberFloat32{Value: resp.Embedding},
				Metadata: &s3vtypes.DocumentMemberString{Value: string(meta)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("put vector: %w", err)
	}

	updateJobStatus(ctx, questionID, "done", retryCount)
	slog.Info("embedding job done", "question_id", questionID)
	return nil
}

func updateJobStatus(ctx context.Context, questionID, status string, retryCount int) {
	_, err := ddbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &embedJobsTable,
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
			"sk":          &types.AttributeValueMemberS{Value: "job"},
		},
		UpdateExpression: aws.String("SET #s = :s, retry_count = :r"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: status},
			":r": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", retryCount)},
		},
	})
	if err != nil {
		slog.Error("update job status", "error", err)
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
	_ = time.Now() // ensure time import used
	lambda.Start(handler)
}
