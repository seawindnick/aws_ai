package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/workshop/wrong-question/internal/model"
)

const (
	maxImageBytes = 10 * 1024 * 1024 // 10 MB
	minImageBytes = 1024              // 1 KB，过小视为无效图片
)

// RecognitionResult 是第三方 API 的原始返回，字段均需校验后使用（R3）。
type RecognitionResult struct {
	RawText    string           `json:"raw_text"`
	Subject    string           `json:"subject"`
	TopicTags  []string         `json:"topic_tags"`
	Category   model.QuestionCategory `json:"category"`
	Confidence float64          `json:"confidence"`
}

type RecognitionService struct {
	apiURL     string
	apiKey     string
	timeoutSec int
}

func NewRecognitionService(apiURL, apiKey string, timeoutSec int) *RecognitionService {
	return &RecognitionService{apiURL: apiURL, apiKey: apiKey, timeoutSec: timeoutSec}
}

// ValidateImageBytes 在调用 API 前进行图片边界检查，返回明确错误不兜底（R2）。
func ValidateImageBytes(data []byte) error {
	size := len(data)
	if size < minImageBytes {
		return fmt.Errorf("image too small (%d bytes), minimum is %d bytes", size, minImageBytes)
	}
	if size > maxImageBytes {
		return fmt.Errorf("image too large (%d bytes), maximum is %d bytes", size, maxImageBytes)
	}
	// 检查常见图片文件头
	if !isJPEG(data) && !isPNG(data) {
		return fmt.Errorf("unsupported image format: only JPEG and PNG are accepted")
	}
	return nil
}

func isJPEG(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
}

func isPNG(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A
}

// Recognize 调用第三方 API 识别题目，返回原始结果供上层按置信度分流。
// 注意：confidence 缺失时按 0 处理（R3 例外约定），其余关键字段缺失返回 error。
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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("recognition api rate limited (429)")
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, fmt.Errorf("recognition api rejected image content (422): image may be unreadable or contain no text")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recognition api returned unexpected status %d", resp.StatusCode)
	}

	var result RecognitionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode recognition response: %w", err)
	}

	// 关键字段校验（R3）
	if result.RawText == "" {
		return nil, fmt.Errorf("recognition api returned empty raw_text")
	}
	if result.Subject == "" {
		return nil, fmt.Errorf("recognition api returned empty subject")
	}
	// confidence 缺失按 0 处理，不报错（spec R3 例外）
	// category 缺失时给 unknown，不报错（允许人工审核时修正）
	if result.Category == "" {
		result.Category = model.CategoryUnknown
	}

	return &result, nil
}
