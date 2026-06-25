package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ddbClient      *dynamodb.Client
	bedrockClient  *bedrockruntime.Client
	s3Client       *s3.Client
	questionsTable string
	visionModelID  string
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
	s3Client = s3.NewFromConfig(cfg)

	questionsTable = mustEnv("DYNAMO_TABLE_QUESTIONS")
	visionModelID = getEnv("VISION_MODEL_ID", "us.anthropic.claude-haiku-4-5-20251001-v1:0")
}

// Question stored in DynamoDB
type Question struct {
	QuestionID string `dynamodbav:"question_id"` // PK
	UserID     string `dynamodbav:"user_id"`     // GSI PK
	ImageKey   string `dynamodbav:"image_key"`
	Subject    string `dynamodbav:"subject"`
	RawText    string `dynamodbav:"raw_text"`
	Category   string `dynamodbav:"category"`
	Analysis   string `dynamodbav:"analysis"`
	KeyPoints  string `dynamodbav:"key_points"`
	Status     string `dynamodbav:"status"` // pending | done | failed
	CreatedAt  string `dynamodbav:"created_at"`
	UpdatedAt  string `dynamodbav:"updated_at"`
}

// bedrockResponse is Claude's Messages API response
type bedrockResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// analysisResult parsed from LLM output
type analysisResult struct {
	Subject   string `json:"subject"`
	Category  string `json:"category"`
	RawText   string `json:"raw_text"`
	Analysis  string `json:"analysis"`
	KeyPoints string `json:"key_points"`
}

func handler(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.URLDecodedKey
		if key == "" {
			key = record.S3.Object.Key
		}

		slog.Info("processing image", "bucket", bucket, "key", key)

		if err := processImage(ctx, bucket, key); err != nil {
			slog.Error("process image failed", "key", key, "error", err)
			// Mark as failed in DynamoDB if we have a question ID from the key
			if qid := questionIDFromKey(key); qid != "" {
				updateStatus(ctx, qid, "failed")
			}
		}
	}
	return nil
}

// processImage downloads the image from S3, sends to Bedrock Claude vision, stores result in DynamoDB
func processImage(ctx context.Context, bucket, key string) error {
	// Download image bytes
	out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("get s3 object: %w", err)
	}
	defer out.Body.Close()

	buf := make([]byte, 0, 1<<20)
	tmp := make([]byte, 32*1024)
	for {
		n, err2 := out.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err2 != nil {
			break
		}
	}

	imgB64 := base64.StdEncoding.EncodeToString(buf)
	mediaType := mediaTypeFromKey(key)

	// Build Claude Messages API payload
	prompt := `You are an educational AI. Analyze this exam question image.
Return a JSON object with these fields:
- subject: the academic subject (e.g. "Math", "Physics", "English", "Chinese", "Biology", "Chemistry", "History", "Geography")
- category: question type ("multiple_choice", "fill_blank", "essay", "true_false", "calculation")
- raw_text: verbatim text of the question extracted from the image
- analysis: detailed explanation of the correct answer and reasoning
- key_points: comma-separated list of knowledge points tested

Respond with valid JSON only, no markdown.`

	payload, _ := json.Marshal(map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        1024,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mediaType,
							"data":       imgB64,
						},
					},
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
	})

	resp, err := bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(visionModelID),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return fmt.Errorf("bedrock invoke: %w", err)
	}

	var br bedrockResponse
	if err := json.Unmarshal(resp.Body, &br); err != nil || len(br.Content) == 0 {
		return fmt.Errorf("decode bedrock response: %w", err)
	}

	var result analysisResult
	if err := json.Unmarshal([]byte(br.Content[0].Text), &result); err != nil {
		// If JSON parse fails, store raw text as analysis
		result.Analysis = br.Content[0].Text
		result.Category = "unknown"
	}

	// Derive question_id and user_id from S3 key
	// Expected key format: images/{user_id}/{question_id}.{ext}
	qid, uid := parseKey(key)

	now := time.Now().UTC().Format(time.RFC3339)
	q := Question{
		QuestionID: qid,
		UserID:     uid,
		ImageKey:   key,
		Subject:    result.Subject,
		RawText:    result.RawText,
		Category:   result.Category,
		Analysis:   result.Analysis,
		KeyPoints:  result.KeyPoints,
		Status:     "done",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	item, err := attributevalue.MarshalMap(q)
	if err != nil {
		return fmt.Errorf("marshal question: %w", err)
	}

	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(questionsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put question: %w", err)
	}

	slog.Info("question analyzed and stored", "question_id", qid, "user_id", uid, "subject", result.Subject)
	return nil
}

func updateStatus(ctx context.Context, questionID, status string) {
	_, err := ddbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(questionsTable),
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
		UpdateExpression: aws.String("SET #s = :s, updated_at = :t"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: status},
			":t": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		slog.Error("update status failed", "question_id", questionID, "error", err)
	}
}

// parseKey extracts user_id and question_id from S3 key.
// Expected format: images/{user_id}/{question_id}.{ext}
func parseKey(key string) (questionID, userID string) {
	// strip leading "images/"
	trimmed := strings.TrimPrefix(key, "images/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return key, "unknown"
	}
	userID = parts[0]
	filename := parts[1]
	// strip extension
	if dot := strings.LastIndex(filename, "."); dot > 0 {
		questionID = filename[:dot]
	} else {
		questionID = filename
	}
	return questionID, userID
}

// questionIDFromKey returns the question ID part from the S3 key for error tracking.
func questionIDFromKey(key string) string {
	qid, _ := parseKey(key)
	return qid
}

func mediaTypeFromKey(key string) string {
	lower := strings.ToLower(key)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
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
