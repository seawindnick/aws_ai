package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RecognitionResult struct {
	RawText   string   `json:"raw_text"`
	Subject   string   `json:"subject"`
	TopicTags []string `json:"topic_tags"`
}

type RecognitionService struct {
	apiURL     string
	apiKey     string
	timeoutSec int
}

func NewRecognitionService(apiURL, apiKey string, timeoutSec int) *RecognitionService {
	return &RecognitionService{apiURL: apiURL, apiKey: apiKey, timeoutSec: timeoutSec}
}

func (s *RecognitionService) Recognize(ctx context.Context, imageBase64 string) (*RecognitionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeoutSec)*time.Second)
	defer cancel()

	body := fmt.Sprintf(`{"image":"%s"}`, imageBase64)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/recognize", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build recognition request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call recognition api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recognition api returned status %d", resp.StatusCode)
	}

	var result RecognitionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode recognition response: %w", err)
	}

	if result.RawText == "" {
		return nil, fmt.Errorf("recognition api returned empty raw_text")
	}
	if result.Subject == "" {
		return nil, fmt.Errorf("recognition api returned empty subject")
	}

	return &result, nil
}
