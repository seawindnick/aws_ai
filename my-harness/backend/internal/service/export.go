package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/workshop/wrong-question/internal/repository"
)

type ExportService struct {
	questionRepo *repository.QuestionRepo
	exportDir    string
}

func NewExportService(repo *repository.QuestionRepo, exportDir string) *ExportService {
	return &ExportService{questionRepo: repo, exportDir: exportDir}
}

func (s *ExportService) ExportPDF(ctx context.Context, userID, subject string) (string, error) {
	questions, err := s.questionRepo.ListByUser(ctx, userID, subject, 1, 200)
	if err != nil {
		return "", fmt.Errorf("fetch questions for export: %w", err)
	}
	if len(questions) == 0 {
		return "", fmt.Errorf("no questions found for export")
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Wrong Question Summary")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)

	for i, q := range questions {
		pdf.MultiCell(0, 8, fmt.Sprintf("%d. [%s] %s", i+1, q.Subject, q.RawText), "", "", false)
		pdf.Ln(4)
	}

	filename := uuid.New().String() + ".pdf"
	outPath := filepath.Join(s.exportDir, filename)

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return "", fmt.Errorf("write pdf file: %w", err)
	}

	return outPath, nil
}
