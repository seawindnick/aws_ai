package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type RecommendItem struct {
	QuestionID string  `json:"question_id"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type bedrockRequest struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	Messages  []bkMessage `json:"messages"`
}

type bkMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type BedrockService struct {
	client     *bedrockruntime.Client
	modelID    string
	timeoutSec int
}

func NewBedrockService(client *bedrockruntime.Client, modelID string, timeoutSec int) *BedrockService {
	return &BedrockService{client: client, modelID: modelID, timeoutSec: timeoutSec}
}

func (s *BedrockService) Recommend(ctx context.Context, errorSummary string) ([]*RecommendItem, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeoutSec)*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(
		"Based on the student's error records below, recommend the top 5 questions to review and explain why.\nReturn JSON array: [{\"question_id\":\"...\",\"reason\":\"...\",\"confidence\":0.0}]\n\n%s",
		errorSummary,
	)

	reqBody, err := json.Marshal(bedrockRequest{
		Model:     s.modelID,
		MaxTokens: 1024,
		Messages:  []bkMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal bedrock request: %w", err)
	}

	out, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.modelID),
		ContentType: aws.String("application/json"),
		Body:        reqBody,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke bedrock model: %w", err)
	}

	var bkResp bedrockResponse
	if err := json.Unmarshal(out.Body, &bkResp); err != nil {
		return nil, fmt.Errorf("decode bedrock response: %w", err)
	}
	if len(bkResp.Content) == 0 || bkResp.Content[0].Text == "" {
		return nil, fmt.Errorf("bedrock returned empty content")
	}

	var items []*RecommendItem
	if err := json.Unmarshal([]byte(bkResp.Content[0].Text), &items); err != nil {
		return nil, fmt.Errorf("parse bedrock recommendation json: %w", err)
	}

	for _, item := range items {
		if item.QuestionID == "" {
			return nil, fmt.Errorf("bedrock recommendation missing question_id")
		}
		if item.Reason == "" {
			return nil, fmt.Errorf("bedrock recommendation missing reason")
		}
		// confidence 缺失时按 0 处理（已由 JSON 零值保证）
	}

	return items, nil
}
