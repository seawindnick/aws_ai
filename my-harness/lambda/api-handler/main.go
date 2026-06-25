package main

import (
	"context"
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
	cognitoSvc "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitoTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3presign "github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ddbClient     *dynamodb.Client
	cognitoClient *cognitoSvc.Client
	s3Client      *s3.Client
	s3PresignCli  *s3presign.PresignClient

	usersTable         string
	questionsTable     string
	reviewSchedulesTbl string
	imagesBucket       string
	cognitoClientID    string
	cognitoUserPoolID  string
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		slog.Error("load aws config", "error", err)
		os.Exit(1)
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
	cognitoClient = cognitoSvc.NewFromConfig(cfg)
	s3Client = s3.NewFromConfig(cfg)
	s3PresignCli = s3presign.NewPresignClient(s3Client)

	usersTable = mustEnv("DYNAMO_TABLE_USERS")
	questionsTable = mustEnv("DYNAMO_TABLE_QUESTIONS")
	reviewSchedulesTbl = mustEnv("DYNAMO_TABLE_REVIEW_SCHEDULES")
	imagesBucket = mustEnv("IMAGES_BUCKET")
	cognitoClientID = mustEnv("COGNITO_CLIENT_ID")
	cognitoUserPoolID = mustEnv("COGNITO_USER_POOL_ID")
}

// User stored in DynamoDB users table
type User struct {
	UserID    string `dynamodbav:"user_id" json:"user_id"`
	Email     string `dynamodbav:"email"   json:"email"`
	Nickname  string `dynamodbav:"nickname" json:"nickname"`
	Role      string `dynamodbav:"role"    json:"role"`
	CreatedAt string `dynamodbav:"created_at" json:"created_at"`
}

// Question stored in DynamoDB questions table
type Question struct {
	QuestionID string `dynamodbav:"question_id" json:"question_id"`
	UserID     string `dynamodbav:"user_id"     json:"user_id"`
	ImageKey   string `dynamodbav:"image_key"   json:"image_key"`
	Subject    string `dynamodbav:"subject"     json:"subject"`
	RawText    string `dynamodbav:"raw_text"    json:"raw_text"`
	Category   string `dynamodbav:"category"    json:"category"`
	Analysis   string `dynamodbav:"analysis"    json:"analysis"`
	KeyPoints  string `dynamodbav:"key_points"  json:"key_points"`
	Status     string `dynamodbav:"status"      json:"status"`
	CreatedAt  string `dynamodbav:"created_at"  json:"created_at"`
	UpdatedAt  string `dynamodbav:"updated_at"  json:"updated_at"`
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RawPath

	slog.Info("request", "method", method, "path", path)

	// Route dispatch
	switch {
	case path == "/api/health" && method == "GET":
		return jsonResp(200, map[string]string{"status": "ok"})

	case path == "/api/auth/signup" && method == "POST":
		return handleSignup(ctx, req)

	case path == "/api/auth/login" && method == "POST":
		return handleLogin(ctx, req)

	case path == "/api/questions" && method == "GET":
		return handleListQuestions(ctx, req)

	case path == "/api/questions/upload-url" && method == "POST":
		return handlePresignUpload(ctx, req)

	case path == "/api/questions" && method == "POST":
		return handleCreateQuestion(ctx, req)

	case strings.HasPrefix(path, "/api/questions/") && method == "GET":
		qid := strings.TrimPrefix(path, "/api/questions/")
		return handleGetQuestion(ctx, req, qid)

	case strings.HasPrefix(path, "/api/questions/") && method == "DELETE":
		qid := strings.TrimPrefix(path, "/api/questions/")
		return handleDeleteQuestion(ctx, req, qid)

	case path == "/api/me" && method == "GET":
		return handleGetMe(ctx, req)

	default:
		return jsonResp(404, map[string]string{"error": "not found"})
	}
}

// handleSignup registers a new user in Cognito and DynamoDB
func handleSignup(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Email == "" || body.Password == "" {
		return jsonResp(400, map[string]string{"error": "email and password required"})
	}

	out, err := cognitoClient.SignUp(ctx, &cognitoSvc.SignUpInput{
		ClientId: aws.String(cognitoClientID),
		Username: aws.String(body.Email),
		Password: aws.String(body.Password),
		UserAttributes: []cognitoTypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String(body.Email)},
		},
	})
	if err != nil {
		return jsonResp(400, map[string]string{"error": err.Error()})
	}

	userID := aws.ToString(out.UserSub)
	now := time.Now().UTC().Format(time.RFC3339)
	nickname := body.Nickname
	if nickname == "" {
		nickname = strings.Split(body.Email, "@")[0]
	}

	user := User{
		UserID:    userID,
		Email:     body.Email,
		Nickname:  nickname,
		Role:      "student",
		CreatedAt: now,
	}
	item, _ := attributevalue.MarshalMap(user)
	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(usersTable),
		Item:      item,
	})
	if err != nil {
		return jsonResp(500, map[string]string{"error": "failed to store user"})
	}

	return jsonResp(201, map[string]string{"user_id": userID, "message": "check email for verification code"})
}

// handleLogin authenticates user via Cognito
func handleLogin(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Email == "" {
		return jsonResp(400, map[string]string{"error": "email and password required"})
	}

	out, err := cognitoClient.InitiateAuth(ctx, &cognitoSvc.InitiateAuthInput{
		AuthFlow: cognitoTypes.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(cognitoClientID),
		AuthParameters: map[string]string{
			"USERNAME": body.Email,
			"PASSWORD": body.Password,
		},
	})
	if err != nil {
		return jsonResp(401, map[string]string{"error": "invalid credentials"})
	}
	if out.AuthenticationResult == nil {
		return jsonResp(401, map[string]string{"error": "authentication failed"})
	}

	return jsonResp(200, map[string]string{
		"access_token":  aws.ToString(out.AuthenticationResult.AccessToken),
		"id_token":      aws.ToString(out.AuthenticationResult.IdToken),
		"refresh_token": aws.ToString(out.AuthenticationResult.RefreshToken),
	})
}

// handlePresignUpload generates a pre-signed S3 PUT URL for image upload
func handlePresignUpload(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	var body struct {
		QuestionID string `json:"question_id"`
		Extension  string `json:"extension"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.QuestionID == "" {
		return jsonResp(400, map[string]string{"error": "question_id required"})
	}

	ext := body.Extension
	if ext == "" {
		ext = "jpg"
	}
	key := fmt.Sprintf("images/%s/%s.%s", userID, body.QuestionID, ext)

	presigned, err := s3PresignCli.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(imagesBucket),
		Key:    aws.String(key),
	}, s3presign.WithPresignExpires(15*time.Minute))
	if err != nil {
		return jsonResp(500, map[string]string{"error": "failed to generate upload URL"})
	}

	return jsonResp(200, map[string]string{
		"upload_url":  presigned.URL,
		"key":         key,
		"question_id": body.QuestionID,
	})
}

// handleCreateQuestion creates a placeholder question record (before image upload triggers analysis)
func handleCreateQuestion(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	var body struct {
		QuestionID string `json:"question_id"`
		Subject    string `json:"subject"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.QuestionID == "" {
		return jsonResp(400, map[string]string{"error": "question_id required"})
	}

	now := time.Now().UTC().Format(time.RFC3339)
	q := Question{
		QuestionID: body.QuestionID,
		UserID:     userID,
		Subject:    body.Subject,
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	item, _ := attributevalue.MarshalMap(q)
	_, err := ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(questionsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(question_id)"),
	})
	if err != nil {
		return jsonResp(409, map[string]string{"error": "question already exists"})
	}

	return jsonResp(201, q)
}

// handleListQuestions returns all questions for the authenticated user
func handleListQuestions(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	out, err := ddbClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(questionsTable),
		IndexName:              aws.String("user_created_index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return jsonResp(500, map[string]string{"error": "query failed"})
	}

	questions := make([]Question, 0, len(out.Items))
	for _, item := range out.Items {
		var q Question
		if err := attributevalue.UnmarshalMap(item, &q); err == nil {
			questions = append(questions, q)
		}
	}

	return jsonResp(200, map[string]any{"items": questions, "count": len(questions)})
}

// handleGetQuestion returns a single question by ID
func handleGetQuestion(ctx context.Context, req events.APIGatewayV2HTTPRequest, questionID string) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	out, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(questionsTable),
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
	})
	if err != nil || out.Item == nil {
		return jsonResp(404, map[string]string{"error": "question not found"})
	}

	var q Question
	if err := attributevalue.UnmarshalMap(out.Item, &q); err != nil {
		return jsonResp(500, map[string]string{"error": "unmarshal error"})
	}

	// Enforce user isolation (R1)
	if q.UserID != userID {
		return jsonResp(403, map[string]string{"error": "forbidden"})
	}

	return jsonResp(200, q)
}

// handleDeleteQuestion deletes a question (user must own it)
func handleDeleteQuestion(ctx context.Context, req events.APIGatewayV2HTTPRequest, questionID string) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	// Verify ownership first
	out, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(questionsTable),
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
	})
	if err != nil || out.Item == nil {
		return jsonResp(404, map[string]string{"error": "question not found"})
	}

	var q Question
	attributevalue.UnmarshalMap(out.Item, &q)
	if q.UserID != userID {
		return jsonResp(403, map[string]string{"error": "forbidden"})
	}

	_, err = ddbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(questionsTable),
		Key: map[string]types.AttributeValue{
			"question_id": &types.AttributeValueMemberS{Value: questionID},
		},
	})
	if err != nil {
		return jsonResp(500, map[string]string{"error": "delete failed"})
	}

	return jsonResp(200, map[string]string{"deleted": questionID})
}

// handleGetMe returns the current user's profile
func handleGetMe(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := userIDFromRequest(req)
	if userID == "" {
		return jsonResp(401, map[string]string{"error": "unauthorized"})
	}

	out, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(usersTable),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil || out.Item == nil {
		return jsonResp(404, map[string]string{"error": "user not found"})
	}

	var user User
	attributevalue.UnmarshalMap(out.Item, &user)
	return jsonResp(200, user)
}

// userIDFromRequest extracts user_id from the Cognito JWT claims injected by API Gateway
func userIDFromRequest(req events.APIGatewayV2HTTPRequest) string {
	// API Gateway v2 with Cognito JWT authorizer puts claims in requestContext.authorizer.jwt.claims
	if claims := req.RequestContext.Authorizer.JWT.Claims; claims != nil {
		if sub, ok := claims["sub"]; ok {
			return sub
		}
	}
	return ""
}

func jsonResp(code int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var " + key + " not set")
	}
	return v
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})))
	lambda.Start(handler)
}
