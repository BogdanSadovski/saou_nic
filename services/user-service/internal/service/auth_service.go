package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/real-ass/user-service/internal/domain"
	jwtpkg "github.com/real-ass/user-service/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

type AuthService struct {
	userRepo   domain.UserRepository
	tokenMgr   *jwtpkg.TokenManager
}

func NewAuthService(userRepo domain.UserRepository, tokenMgr *jwtpkg.TokenManager) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		tokenMgr: tokenMgr,
	}
}

func (s *AuthService) Register(ctx context.Context, req domain.CreateUserRequest) (*jwtpkg.Tokens, error) {
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		Provider:     domain.ProviderLocal,
		EmailVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	tokens, err := s.tokenMgr.GenerateTokens(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*jwtpkg.Tokens, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail the login
		fmt.Printf("failed to update last login: %v\n", err)
	}

	tokens, err := s.tokenMgr.GenerateTokens(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*jwtpkg.Tokens, error) {
	claims, err := s.tokenMgr.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	tokens, err := s.tokenMgr.GenerateTokens(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *AuthService) ValidateToken(token string) (*jwtpkg.Claims, error) {
	return s.tokenMgr.ValidateToken(token)
}

func (s *AuthService) GetUserID(token string) (uuid.UUID, error) {
	claims, err := s.tokenMgr.ValidateToken(token)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}
