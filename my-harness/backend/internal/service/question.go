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
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type QuestionService struct {
	repo        *repository.QuestionRepo
	recognition *RecognitionService
	imageDir    string
}

func NewQuestionService(repo *repository.QuestionRepo, recognition *RecognitionService, imageDir string) *QuestionService {
	return &QuestionService{repo: repo, recognition: recognition, imageDir: imageDir}
}

func (s *QuestionService) Upload(ctx context.Context, userID string, file multipart.File) (*model.Question, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read image file: %w", err)
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
	result, err := s.recognition.Recognize(ctx, encoded)
	if err != nil {
		// 识别失败不兜底（R2），删除已写入的图片，向上返回错误
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("recognize question: %w", err)
	}

	q := &model.Question{
		ID:        questionID,
		UserID:    userID,
		ImagePath: imagePath,
		RawText:   result.RawText,
		Subject:   result.Subject,
		TopicTags: result.TopicTags,
		Source:    "third_party_api",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, q); err != nil {
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("save question: %w", err)
	}

	return q, nil
}

func (s *QuestionService) List(ctx context.Context, userID, subject string, page, pageSize int) ([]*model.Question, error) {
	questions, err := s.repo.ListByUser(ctx, userID, subject, page, pageSize)
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
