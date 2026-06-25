package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
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
	tagRepo     *repository.TagRepo
	recognition *RecognitionService
	embedJobs   *EmbeddingJobService
	s3vRepo     *repository.S3VectorsRepo
	reviewRepo  *repository.ReviewRepo
	imageDir    string
	cfg         *config.Config
}

func NewQuestionService(
	repo *repository.QuestionRepo,
	tagRepo *repository.TagRepo,
	recognition *RecognitionService,
	embedJobs *EmbeddingJobService,
	s3vRepo *repository.S3VectorsRepo,
	reviewRepo *repository.ReviewRepo,
	cfg *config.Config,
) *QuestionService {
	return &QuestionService{
		repo:       repo,
		tagRepo:    tagRepo,
		recognition: recognition,
		embedJobs:  embedJobs,
		s3vRepo:    s3vRepo,
		reviewRepo: reviewRepo,
		imageDir:   cfg.ImageDir,
		cfg:        cfg,
	}
}

// UploadResult 单张图片上传结果。
type UploadResult struct {
	Question      *model.Question
	NeedsReview   bool
	StatusMessage string
}

// BatchUploadResult 批量上传结果（REQ-UPLOAD-10）。
type BatchUploadResult struct {
	Succeeded []*UploadResult
	Failed    []BatchUploadError
}

type BatchUploadError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

func (s *QuestionService) Upload(ctx context.Context, userID string, file multipart.File) (*UploadResult, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read image file: %w", err)
	}
	return s.uploadBytes(ctx, userID, data)
}

// BatchUpload 逐个处理，每张独立成功或失败（REQ-UPLOAD-10）。
func (s *QuestionService) BatchUpload(ctx context.Context, userID string, files []multipart.File) *BatchUploadResult {
	result := &BatchUploadResult{}
	for i, file := range files {
		data, err := io.ReadAll(file)
		if err != nil {
			result.Failed = append(result.Failed, BatchUploadError{Index: i, Error: fmt.Sprintf("read image: %s", err)})
			continue
		}
		res, err := s.uploadBytes(ctx, userID, data)
		if err != nil {
			result.Failed = append(result.Failed, BatchUploadError{Index: i, Error: err.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, res)
	}
	return result
}

func (s *QuestionService) uploadBytes(ctx context.Context, userID string, data []byte) (*UploadResult, error) {
	if err := ValidateImageBytes(data); err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	questionID := uuid.New().String()
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
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("recognize question: %w", err)
	}

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
		Source:     "third_party_api",
		Status:     status,
		Category:   recognition.Category,
		Confidence: recognition.Confidence,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, q); err != nil {
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("save question: %w", err)
	}

	// 将识别出的 topic_tags 存为 suggested 标签
	if len(recognition.TopicTags) > 0 {
		if err := s.createSuggestedTags(ctx, questionID, userID, recognition.TopicTags); err != nil {
			return nil, fmt.Errorf("save suggested tags: %w", err)
		}
	}

	// 异步触发向量生成（写入 embedding_jobs）
	if s.embedJobs != nil {
		if err := s.embedJobs.Enqueue(ctx, questionID, userID); err != nil {
			slog.Warn("enqueue embedding job", "question_id", questionID, "error", err)
		}
	}

	return &UploadResult{
		Question:      q,
		NeedsReview:   status == model.QuestionStatusPendingReview,
		StatusMessage: statusMsg,
	}, nil
}

func (s *QuestionService) createSuggestedTags(ctx context.Context, questionID, userID string, names []string) error {
	for _, name := range names {
		if name == "" {
			continue
		}
		t := &model.Tag{
			ID:         uuid.New().String(),
			QuestionID: questionID,
			UserID:     userID,
			Name:       name,
			Status:     model.TagStatusSuggested,
			CreatedAt:  time.Now().UTC(),
		}
		if err := s.tagRepo.Create(ctx, t); err != nil {
			return fmt.Errorf("insert suggested tag %q: %w", name, err)
		}
	}
	return nil
}

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

// Search 多维搜索（REQ-SEARCH-01 ~ 06）。
// role 为调用者角色；student 强制只可见 approved 题目（REQ-REVIEW-01）。
func (s *QuestionService) Search(ctx context.Context, params model.SearchParams, role string) ([]*model.Question, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if role == string(model.RoleStudent) {
		params.Status = model.QuestionStatusApproved
	}
	questions, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search questions: %w", err)
	}
	return questions, nil
}

// List 题目列表（REQ-REVIEW-01）。student 强制只可见 approved 题目。
func (s *QuestionService) List(ctx context.Context, userID, subject string, status model.QuestionStatus, page, pageSize int, role string) ([]*model.Question, error) {
	if role == string(model.RoleStudent) {
		status = model.QuestionStatusApproved
	}
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

	// cascade MySQL: tags, paper_questions, error_records, review_records
	if err := s.tagRepo.DeleteByQuestion(ctx, id); err != nil {
		return fmt.Errorf("cascade delete tags: %w", err)
	}
	if err := s.repo.DeleteCascadeRelated(ctx, id); err != nil {
		return fmt.Errorf("cascade delete related: %w", err)
	}

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete question record: %w", err)
	}

	// best-effort: DynamoDB review schedule
	if err := s.reviewRepo.DeleteSchedule(ctx, userID, id); err != nil {
		slog.Warn("delete review schedule", "question_id", id, "error", err)
	}

	// best-effort: DynamoDB embedding job
	if s.embedJobs != nil {
		if err := s.embedJobs.Delete(ctx, id); err != nil {
			slog.Warn("delete embedding job", "question_id", id, "error", err)
		}
	}

	// best-effort: S3 Vectors
	if s.s3vRepo != nil {
		if err := s.s3vRepo.Delete(ctx, id); err != nil {
			slog.Warn("delete s3 vector", "question_id", id, "error", err)
		}
	}

	// best-effort: image file
	if err := os.Remove(q.ImagePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("delete image file", "path", q.ImagePath, "error", err)
	}

	return nil
}

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
