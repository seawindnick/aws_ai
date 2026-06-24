package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type ClassService struct {
	classRepo *repository.ClassRepo
}

func NewClassService(classRepo *repository.ClassRepo) *ClassService {
	return &ClassService{classRepo: classRepo}
}

func (s *ClassService) Create(ctx context.Context, teacherID, name string) (*model.Class, error) {
	code, err := s.generateUniqueCode(ctx)
	if err != nil {
		return nil, err
	}
	c := &model.Class{
		ID:         uuid.New().String(),
		Name:       name,
		TeacherID:  teacherID,
		InviteCode: code,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.classRepo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create class: %w", err)
	}
	return c, nil
}

func (s *ClassService) ListMy(ctx context.Context, userID, role string) ([]*model.Class, error) {
	if role == string(model.RoleTeacher) || role == string(model.RoleAdmin) {
		return s.classRepo.ListByTeacher(ctx, userID)
	}
	return s.classRepo.ListByMember(ctx, userID)
}

func (s *ClassService) Get(ctx context.Context, classID, userID, role string) (*model.Class, error) {
	c, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("class not found")
	}
	if role == string(model.RoleTeacher) && c.TeacherID != userID {
		return nil, fmt.Errorf("forbidden")
	}
	return c, nil
}

func (s *ClassService) Join(ctx context.Context, userID, inviteCode string) (*model.Class, error) {
	c, err := s.classRepo.GetByInviteCode(ctx, inviteCode)
	if err != nil || c == nil {
		return nil, fmt.Errorf("invalid invite code")
	}
	member := &model.ClassMember{
		ID:       uuid.New().String(),
		ClassID:  c.ID,
		UserID:   userID,
		JoinedAt: time.Now().UTC(),
	}
	if err := s.classRepo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("join class: %w", err)
	}
	return c, nil
}

func (s *ClassService) Leave(ctx context.Context, classID, userID string) error {
	return s.classRepo.RemoveMember(ctx, classID, userID)
}

func (s *ClassService) RemoveMember(ctx context.Context, classID, teacherID, targetUserID string) error {
	c, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		return fmt.Errorf("class not found")
	}
	if c.TeacherID != teacherID {
		return fmt.Errorf("forbidden")
	}
	return s.classRepo.RemoveMember(ctx, classID, targetUserID)
}

func (s *ClassService) ResetCode(ctx context.Context, classID, teacherID string) (string, error) {
	c, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		return "", fmt.Errorf("class not found")
	}
	if c.TeacherID != teacherID {
		return "", fmt.Errorf("forbidden")
	}
	code, err := s.generateUniqueCode(ctx)
	if err != nil {
		return "", err
	}
	if err := s.classRepo.UpdateInviteCode(ctx, classID, code); err != nil {
		return "", err
	}
	return code, nil
}

func (s *ClassService) ListMembers(ctx context.Context, classID, teacherID string) ([]*model.ClassMember, error) {
	c, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("class not found")
	}
	if c.TeacherID != teacherID {
		return nil, fmt.Errorf("forbidden")
	}
	return s.classRepo.ListMembers(ctx, classID)
}

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (s *ClassService) generateUniqueCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		b := make([]byte, 6)
		for j := range b {
			b[j] = codeChars[rand.Intn(len(codeChars))]
		}
		code := string(b)
		existing, err := s.classRepo.GetByInviteCode(ctx, code)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique invite code")
}
