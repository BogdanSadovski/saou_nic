package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	// The frontend sends `username = fullName`, but full names are not
	// unique (two "Иван Иванов" would collide on users_username_key).
	// Treat the incoming value as a display hint and use the email
	// local-part as the canonical username — it's unique by the email
	// constraint, and we keep the original full name in first/last for
	// later editing.
	username, firstName, lastName := splitNameAndUsername(req.Username, req.Email)

	user := &domain.User{
		Email:        req.Email,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		Provider:     domain.ProviderLocal,
		EmailVerified: false,
	}

	createErr := s.userRepo.Create(ctx, user)
	if createErr != nil && isUniqueViolation(createErr) {
		// If the canonical username (email local-part) is already taken
		// by an unrelated user, append a short random suffix once and
		// retry. After a single retry surface the conflict cleanly.
		user.Username = appendRandomSuffix(username)
		createErr = s.userRepo.Create(ctx, user)
	}
	if createErr != nil {
		if isUniqueViolation(createErr) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", createErr)
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

// splitNameAndUsername reads the inbound full-name / email and returns
// a canonical (username, firstName, lastName) tuple:
//   - username is derived from the email local-part (sanitised), since
//     full names collide and emails are already unique-checked.
//   - first/last name fall out of whitespace-splitting the full name.
//
// If the caller didn't pass a full name we leave both name fields empty.
func splitNameAndUsername(fullName, email string) (username, first, last string) {
	username = sanitiseUsername(localPart(email))
	if username == "" {
		// Pathological case: empty/garbage email → fall back to a
		// random-looking 8-char username so we never persist "".
		username = "user-" + uuid.NewString()[:8]
	}

	parts := strings.Fields(strings.TrimSpace(fullName))
	switch len(parts) {
	case 0:
		// no display name supplied
	case 1:
		first = parts[0]
	default:
		first = parts[0]
		last = strings.Join(parts[1:], " ")
	}
	return username, first, last
}

func localPart(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return email
	}
	return email[:at]
}

func sanitiseUsername(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	out := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out.WriteRune(r)
		}
	}
	return out.String()
}

func appendRandomSuffix(base string) string {
	suffix := uuid.NewString()[:6]
	if base == "" {
		return "user-" + suffix
	}
	return base + "-" + suffix
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505")
}
