package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type AdminService struct {
	userRepo   *repository.UserRepo
	cognito    *cognitoidentityprovider.Client
	clientID   string
	userPoolID string
}

func NewAdminService(
	userRepo *repository.UserRepo,
	cognito *cognitoidentityprovider.Client,
	clientID, userPoolID string,
) *AdminService {
	return &AdminService{userRepo: userRepo, cognito: cognito, clientID: clientID, userPoolID: userPoolID}
}

func (s *AdminService) CreateUser(ctx context.Context, email string, role model.Role) (*model.User, string, error) {
	tempPassword := generatePassword()

	out, err := s.cognito.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:        aws.String(s.userPoolID),
		Username:          aws.String(email),
		TemporaryPassword: aws.String(tempPassword),
		UserAttributes: []cognitotypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
		},
		MessageAction: cognitotypes.MessageActionTypeSuppress,
	})
	if err != nil {
		return nil, "", fmt.Errorf("cognito create user: %w", err)
	}

	sub := ""
	for _, attr := range out.User.Attributes {
		if aws.ToString(attr.Name) == "sub" {
			sub = aws.ToString(attr.Value)
			break
		}
	}

	u := &model.User{
		ID:         uuid.New().String(),
		CognitoSub: sub,
		Email:      email,
		Role:       role,
		Status:     model.UserStatusActive,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, "", fmt.Errorf("create user in db: %w", err)
	}
	return u, tempPassword, nil
}

type ImportResult struct {
	Succeeded []string `json:"succeeded"`
	Failed    []struct {
		Row    int    `json:"row"`
		Email  string `json:"email"`
		Reason string `json:"reason"`
	} `json:"failed"`
}

func (s *AdminService) ImportCSV(ctx context.Context, r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) > 201 { // header + max 200 rows
		return nil, fmt.Errorf("csv exceeds 200 rows limit")
	}

	result := &ImportResult{}
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 2 {
			result.Failed = append(result.Failed, struct {
				Row    int    `json:"row"`
				Email  string `json:"email"`
				Reason string `json:"reason"`
			}{Row: i, Email: "", Reason: "row must have email and role columns"})
			continue
		}
		email, roleStr := row[0], row[1]
		role := model.Role(roleStr)
		if role != model.RoleStudent && role != model.RoleTeacher && role != model.RoleAdmin {
			result.Failed = append(result.Failed, struct {
				Row    int    `json:"row"`
				Email  string `json:"email"`
				Reason string `json:"reason"`
			}{Row: i, Email: email, Reason: "invalid role"})
			continue
		}
		_, _, err := s.CreateUser(ctx, email, role)
		if err != nil {
			result.Failed = append(result.Failed, struct {
				Row    int    `json:"row"`
				Email  string `json:"email"`
				Reason string `json:"reason"`
			}{Row: i, Email: email, Reason: err.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, email)
	}
	return result, nil
}

func (s *AdminService) ListUsers(ctx context.Context, role, status string, page, pageSize int) ([]*model.User, error) {
	return s.userRepo.List(ctx, role, status, page, pageSize)
}

func (s *AdminService) SetStatus(ctx context.Context, userID string, status model.UserStatus) error {
	var deactivatedAt *time.Time
	if status == model.UserStatusInactive {
		now := time.Now().UTC()
		deactivatedAt = &now
		_, _ = s.cognito.AdminDisableUser(ctx, &cognitoidentityprovider.AdminDisableUserInput{
			UserPoolId: aws.String(s.userPoolID),
			Username:   aws.String(userID),
		})
	} else {
		_, _ = s.cognito.AdminEnableUser(ctx, &cognitoidentityprovider.AdminEnableUserInput{
			UserPoolId: aws.String(s.userPoolID),
			Username:   aws.String(userID),
		})
	}
	return s.userRepo.UpdateStatus(ctx, userID, status, deactivatedAt)
}

func (s *AdminService) SetRole(ctx context.Context, userID string, role model.Role) error {
	return s.userRepo.UpdateRole(ctx, userID, role)
}

func generatePassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
