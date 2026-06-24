package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/config"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type QuestionService struct {
	repo        *repository.QuestionRepo
	recognition *RecognitionService
	imageDir    string
	cfg         *config.Config
}

func NewQuestionService(repo *repository.QuestionRepo, recognition *RecognitionService, cfg *config.Config) *QuestionService {
	return &QuestionService{repo: repo, recognition: recognition, imageDir: cfg.ImageDir, cfg: cfg}
}

// UploadResult 上传结果，包含题目本体和状态说明。
type UploadResult struct {
	Question      *model.Question
	NeedsReview   bool
	StatusMessage string
}

func (s *QuestionService) Upload(ctx context.Context, userID string, file multipart.File) (*UploadResult, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read image file: %w", err)
	}

	// 图片边界校验（格式、大小）
	if err := ValidateImageBytes(data); err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	questionID := uuid.New().String()

	// 路径由服务端生成，禁止拼接用户输入（R7）
	dir := filepath.Join(s.imageDir, userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create image dir: %w", err)
	}
	imagePath := filepath.Join(dir, questionID+".jpg")
	if err := os.WriteFile(imagePath, data, 0644); err != nil {
		return nil, fmt.Errorf("write image file: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	recognition, err := s.recognition.Recognize(ctx, encoded)
	if err != nil {
		// 识别失败不兜底（R2），删除已写入的图片，向上返回错误
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("recognize question: %w", err)
	}

	// 按置信度分流：高置信度自动通过，中等转人工审核，低置信度拒绝
	status, statusMsg, err := s.classifyByConfidence(recognition.Confidence)
	if err != nil {
		_ = os.Remove(imagePath)
		return nil, err
	}

	q := &model.Question{
		ID:         questionID,
		UserID:     userID,
		ImagePath:  imagePath,
		RawText:    recognition.RawText,
		Subject:    recognition.Subject,
		TopicTags:  recognition.TopicTags,
		Category:   recognition.Category,
		Source:     "third_party_api",
		Status:     status,
		Confidence: recognition.Confidence,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, q); err != nil {
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("save question: %w", err)
	}

	return &UploadResult{
		Question:    q,
		NeedsReview: status == model.QuestionStatusPendingReview,
		StatusMessage: statusMsg,
	}, nil
}

// classifyByConfidence 根据置信度决定题目状态，返回明确错误不静默降级（R2）。
func (s *QuestionService) classifyByConfidence(confidence float64) (model.QuestionStatus, string, error) {
	switch {
	case confidence >= s.cfg.ConfidenceAutoApprove:
		return model.QuestionStatusApproved, "识别置信度高，已自动入库", nil
	case confidence >= s.cfg.ConfidenceMinAccept:
		return model.QuestionStatusPendingReview,
			fmt.Sprintf("识别置信度偏低（%.0f%%），已提交人工审核队列", confidence*100),
			nil
	default:
		return "", "", fmt.Errorf(
			"recognition confidence too low (%.2f < %.2f), image may be blurry or contain non-question content",
			confidence, s.cfg.ConfidenceMinAccept,
		)
	}
}

func (s *QuestionService) List(ctx context.Context, userID, subject string, status model.QuestionStatus, page, pageSize int) ([]*model.Question, error) {
	questions, err := s.repo.ListByUser(ctx, userID, subject, status, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	return questions, nil
}

func (s *QuestionService) Get(ctx context.Context, id, userID string) (*model.Question, error) {
	q, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get question: %w", err)
	}
	return q, nil
}

func (s *QuestionService) Delete(ctx context.Context, id, userID string) error {
	q, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("get question for delete: %w", err)
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete question record: %w", err)
	}
	if err := os.Remove(q.ImagePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete image file: %w", err)
	}
	return nil
}

// UpdateCategory 学生手动修正题目分类。
func (s *QuestionService) UpdateCategory(ctx context.Context, id, userID string, category model.QuestionCategory) error {
	if !isValidCategory(category) {
		return fmt.Errorf("invalid category: %s", category)
	}
	if err := s.repo.UpdateCategory(ctx, id, userID, category); err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

func isValidCategory(c model.QuestionCategory) bool {
	switch c {
	case model.CategoryMultipleChoice, model.CategoryFillBlank, model.CategoryEssay,
		model.CategoryTrueFalse, model.CategoryCalculation, model.CategoryUnknown:
		return true
	}
	return false
}
