package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type AuthHandler struct {
	cognito    *cognitoidentityprovider.Client
	clientID   string
	userPoolID string
	userRepo   *repository.UserRepo
}

func NewAuthHandler(
	cognito *cognitoidentityprovider.Client,
	clientID, userPoolID string,
	userRepo *repository.UserRepo,
) *AuthHandler {
	return &AuthHandler{cognito: cognito, clientID: clientID, userPoolID: userPoolID, userRepo: userRepo}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		SchoolName string `json:"school_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Email == "" || body.Password == "" {
		WriteError(w, apperr.BadRequest("email and password required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := h.cognito.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(h.clientID),
		Username: aws.String(body.Email),
		Password: aws.String(body.Password),
		UserAttributes: []cognitotypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String(body.Email)},
		},
	})
	if err != nil {
		WriteError(w, fmt.Errorf("cognito signup: %w", err))
		return
	}

	user := &model.User{
		ID:         uuid.New().String(),
		CognitoSub: aws.ToString(out.UserSub),
		Email:      body.Email,
		Role:       model.RoleStudent,
		SchoolName: body.SchoolName,
		CreatedAt:  time.Now().UTC(),
	}
	if err := h.userRepo.Create(r.Context(), user); err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]string{"user_id": user.ID})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Email == "" || body.Password == "" {
		WriteError(w, apperr.BadRequest("email and password required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := h.cognito.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: cognitotypes.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(h.clientID),
		AuthParameters: map[string]string{
			"USERNAME": body.Email,
			"PASSWORD": body.Password,
		},
	})
	if err != nil {
		WriteError(w, apperr.New(http.StatusUnauthorized, "invalid credentials"))
		return
	}
	if out.AuthenticationResult == nil {
		WriteError(w, fmt.Errorf("cognito returned nil authentication result"))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"access_token":  aws.ToString(out.AuthenticationResult.AccessToken),
		"refresh_token": aws.ToString(out.AuthenticationResult.RefreshToken),
		"id_token":      aws.ToString(out.AuthenticationResult.IdToken),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.RefreshToken == "" {
		WriteError(w, apperr.BadRequest("refresh_token required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := h.cognito.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: cognitotypes.AuthFlowTypeRefreshTokenAuth,
		ClientId: aws.String(h.clientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": body.RefreshToken,
		},
	})
	if err != nil {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	if out.AuthenticationResult == nil {
		WriteError(w, fmt.Errorf("cognito returned nil authentication result on refresh"))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"access_token": aws.ToString(out.AuthenticationResult.AccessToken),
		"id_token":     aws.ToString(out.AuthenticationResult.IdToken),
	})
}
