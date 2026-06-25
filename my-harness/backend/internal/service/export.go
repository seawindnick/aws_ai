package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type ExportService struct {
	questionRepo *repository.QuestionRepo
	paperRepo    *repository.PaperRepo
	exportDir    string
}

func NewExportService(questionRepo *repository.QuestionRepo, paperRepo *repository.PaperRepo, exportDir string) *ExportService {
	return &ExportService{questionRepo: questionRepo, paperRepo: paperRepo, exportDir: exportDir}
}

// ExportByFilter 按筛选条件导出 PDF（原有功能）。
func (s *ExportService) ExportByFilter(ctx context.Context, userID, subject string) (string, error) {
	params := model.SearchParams{
		UserID:   userID,
		Subject:  subject,
		Status:   model.QuestionStatusApproved,
		Page:     1,
		PageSize: 200,
	}
	questions, err := s.questionRepo.Search(ctx, params)
	if err != nil {
		return "", fmt.Errorf("fetch questions for export: %w", err)
	}
	if len(questions) == 0 {
		return "", fmt.Errorf("no approved questions found for export")
	}
	outPath := filepath.Join(s.exportDir, userID+"-filter.pdf")
	if err := s.renderPDF(questions, "Wrong Question Summary", outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

// ExportPaper 按试卷导出 PDF，跳过非 approved 题目并报告（REQ-PAPER-07）。
// The generated file is stored as <exportDir>/<paperID>.pdf (deterministic) so
// GET /api/papers/download?id=<paperID> can locate it without a DB lookup.
func (s *ExportService) ExportPaper(ctx context.Context, paperID, userID string) (string, []string, error) {
	items, err := s.paperRepo.ListQuestions(ctx, paperID, userID)
	if err != nil {
		return "", nil, fmt.Errorf("list paper questions: %w", err)
	}
	if len(items) == 0 {
		return "", nil, fmt.Errorf("paper contains no questions")
	}

	var approved []*model.Question
	var skipped []string
	for _, item := range items {
		if item.Question.Status == model.QuestionStatusApproved {
			approved = append(approved, item.Question)
		} else {
			skipped = append(skipped, item.QuestionID)
		}
	}
	if len(approved) == 0 {
		return "", skipped, fmt.Errorf("no approved questions in paper; all %d were skipped", len(items))
	}

	paper, err := s.paperRepo.GetByID(ctx, paperID, userID)
	if err != nil {
		return "", nil, fmt.Errorf("get paper: %w", err)
	}

	outPath := filepath.Join(s.exportDir, paperID+".pdf")
	if err := s.renderPDF(approved, paper.Title, outPath); err != nil {
		return "", skipped, err
	}
	return outPath, skipped, nil
}

// GetPaperDownloadPath verifies ownership and returns the path of the previously
// generated PDF. Returns an error if the paper doesn't belong to userID or the
// file hasn't been generated yet.
func (s *ExportService) GetPaperDownloadPath(ctx context.Context, paperID, userID string) (string, error) {
	if _, err := s.paperRepo.GetByID(ctx, paperID, userID); err != nil {
		return "", err
	}
	path := filepath.Join(s.exportDir, paperID+".pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("pdf not found: export the paper first")
	}
	return path, nil
}

func (s *ExportService) renderPDF(questions []*model.Question, title, outPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, title)
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)

	for i, q := range questions {
		line := fmt.Sprintf("%d. [%s][%s] %s", i+1, q.Subject, string(q.Category), q.RawText)
		pdf.MultiCell(0, 8, line, "", "", false)
		pdf.Ln(3)
	}

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return fmt.Errorf("write pdf file: %w", err)
	}
	return nil
}
