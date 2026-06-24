package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type MeService struct {
	userRepo   *repository.UserRepo
	cognito    *cognitoidentityprovider.Client
	clientID   string
	userPoolID string
}

func NewMeService(
	userRepo *repository.UserRepo,
	cognito *cognitoidentityprovider.Client,
	clientID, userPoolID string,
) *MeService {
	return &MeService{userRepo: userRepo, cognito: cognito, clientID: clientID, userPoolID: userPoolID}
}

func (s *MeService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	return u, nil
}

func (s *MeService) UpdateNickname(ctx context.Context, userID, nickname string) error {
	if len(nickname) < 1 || len(nickname) > 50 {
		return fmt.Errorf("nickname must be 1–50 characters")
	}
	return s.userRepo.UpdateNickname(ctx, userID, nickname)
}

func (s *MeService) ChangePassword(ctx context.Context, accessToken, oldPassword, newPassword string) error {
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return fmt.Errorf("new password must be 8–72 characters")
	}
	_, err := s.cognito.ChangePassword(ctx, &cognitoidentityprovider.ChangePasswordInput{
		AccessToken:      aws.String(accessToken),
		PreviousPassword: aws.String(oldPassword),
		ProposedPassword: aws.String(newPassword),
	})
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	return nil
}

func (s *MeService) Deactivate(ctx context.Context, userID, accessToken, password string) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// verify current password via Cognito
	_, err = s.cognito.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: cognitotypes.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(s.clientID),
		AuthParameters: map[string]string{
			"USERNAME": u.Email,
			"PASSWORD": password,
		},
	})
	if err != nil {
		return fmt.Errorf("invalid password")
	}

	now := time.Now().UTC()
	if err := s.userRepo.UpdateStatus(ctx, userID, model.UserStatusInactive, &now); err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}

	// revoke all sessions
	_, _ = s.cognito.GlobalSignOut(ctx, &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessToken),
	})
	return nil
}
