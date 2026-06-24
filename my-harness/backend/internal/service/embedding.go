package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type EmbeddingService struct {
	client     *bedrockruntime.Client
	modelID    string
	timeoutSec int
}

func NewEmbeddingService(client *bedrockruntime.Client, modelID string, timeoutSec int) *EmbeddingService {
	return &EmbeddingService{client: client, modelID: modelID, timeoutSec: timeoutSec}
}

func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeoutSec)*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]string{"inputText": text})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	out, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke embedding model: %w", err)
	}

	var resp struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(resp.Embedding) == 0 {
		return nil, fmt.Errorf("embedding service returned empty vector")
	}
	return resp.Embedding, nil
}
