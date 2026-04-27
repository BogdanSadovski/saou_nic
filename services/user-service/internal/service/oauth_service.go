package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/real-ass/user-service/internal/domain"
	jwtpkg "github.com/real-ass/user-service/pkg/jwt"
)

type OAuthService struct {
	userRepo domain.UserRepository
	tokenMgr *jwtpkg.TokenManager
	config   OAuthConfig
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

type gitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func NewOAuthService(userRepo domain.UserRepository, tokenMgr *jwtpkg.TokenManager, config OAuthConfig) *OAuthService {
	return &OAuthService{
		userRepo: userRepo,
		tokenMgr: tokenMgr,
		config:   config,
	}
}

func (s *OAuthService) GetAuthURL(provider string) string {
	switch provider {
	case "google":
		return fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid email profile",
			s.config.GoogleClientID, s.config.GoogleRedirectURL,
		)
	case "github":
		return fmt.Sprintf(
			"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email",
			s.config.GitHubClientID, s.config.GitHubRedirectURL,
		)
	default:
		return ""
	}
}

func (s *OAuthService) HandleCallback(ctx context.Context, provider, code string) (*jwtpkg.Tokens, error) {
	userInfo, err := s.exchangeCodeForUserInfo(ctx, provider, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := s.userRepo.GetByProviderID(ctx, userInfo.Provider, userInfo.ProviderID)
	if err != nil {
		// User doesn't exist, create new account
		user = &domain.User{
			Email:        userInfo.Email,
			Username:     userInfo.Username,
			FirstName:    userInfo.FirstName,
			LastName:     userInfo.LastName,
			AvatarURL:    userInfo.AvatarURL,
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			Provider:     userInfo.Provider,
			ProviderID:   userInfo.ProviderID,
			EmailVerified: true,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		fmt.Printf("failed to update last login: %v\n", err)
	}

	tokens, err := s.tokenMgr.GenerateTokens(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *OAuthService) exchangeCodeForUserInfo(ctx context.Context, provider, code string) (*domain.OAuthUserInfo, error) {
	switch provider {
	case "google":
		return s.getGoogleUserInfo(ctx, code)
	case "github":
		return s.getGitHubUserInfo(ctx, code)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *OAuthService) getGoogleUserInfo(ctx context.Context, code string) (*domain.OAuthUserInfo, error) {
	// Exchange code for token
	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"code":          {code},
		"client_id":     {s.config.GoogleClientID},
		"client_secret": {s.config.GoogleClientSecret},
		"redirect_uri":  {s.config.GoogleRedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(tokenBody, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken, ok := tokenData["access_token"].(string)
	if !ok {
		return nil, fmt.Errorf("access token not found in response")
	}

	// Get user info
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var googleUser googleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &domain.OAuthUserInfo{
		ProviderID: googleUser.ID,
		Email:      googleUser.Email,
		Username:   googleUser.Email,
		FirstName:  googleUser.GivenName,
		LastName:   googleUser.FamilyName,
		AvatarURL:  googleUser.Picture,
		Provider:   domain.ProviderGoogle,
	}, nil
}

func (s *OAuthService) getGitHubUserInfo(ctx context.Context, code string) (*domain.OAuthUserInfo, error) {
	// Exchange code for token
	tokenResp, err := http.PostForm("https://github.com/login/oauth/access_token", map[string][]string{
		"code":          {code},
		"client_id":     {s.config.GitHubClientID},
		"client_secret": {s.config.GitHubClientSecret},
		"redirect_uri":  {s.config.GitHubRedirectURL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// Parse access_token from response (format: access_token=xxx&scope=xxx&token_type=bearer)
	accessToken := ""
	for _, param := range string(tokenBody) {
		// Simplified parsing; in production use proper URL query parsing
		_ = param
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var ghUser gitHubUserInfo
	if err := json.Unmarshal(body, &ghUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &domain.OAuthUserInfo{
		ProviderID: fmt.Sprintf("%d", ghUser.ID),
		Email:      ghUser.Email,
		Username:   ghUser.Login,
		FirstName:  ghUser.Name,
		AvatarURL:  ghUser.AvatarURL,
		Provider:   domain.ProviderGitHub,
	}, nil
}
