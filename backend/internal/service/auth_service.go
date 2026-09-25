package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailRequired      = errors.New("email is required")
	ErrPasswordRequired   = errors.New("password is required")
	ErrInvalidEmailFormat = errors.New("invalid email format")
)

// LoginResponse represents the payload returned upon successful authentication.
type LoginResponse struct {
	AccessToken string             `json:"access_token"`
	TokenType   string             `json:"token_type"`
	ExpiresIn   int64              `json:"expires_in"`
	User        model.UserResponse `json:"user"`
}

// AuthService defines high-level authentication and user session operations.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	Logout(ctx context.Context, tokenString string) error
	GetCurrentUser(ctx context.Context, userID string) (*model.UserResponse, error)
}

type authService struct {
	userRepo         repository.UserRepository
	revokedTokenRepo repository.RevokedTokenRepository
	jwtService       JWTService
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	revokedTokenRepo repository.RevokedTokenRepository,
	jwtService JWTService,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		revokedTokenRepo: revokedTokenRepo,
		jwtService:       jwtService,
	}
}

// Login verifies credentials and generates a real JWT token.
func (s *authService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, ErrEmailRequired
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, ErrInvalidEmailFormat
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify bcrypt password hash
	if !user.CheckPassword(password) {
		return nil, ErrInvalidCredentials
	}

	// Generate real signed JWT
	tokenString, expiresIn, err := s.jwtService.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        user.ToResponse(),
	}, nil
}

// Logout revokes the provided JWT on the server side.
func (s *authService) Logout(ctx context.Context, tokenString string) error {
	if tokenString == "" {
		return nil
	}

	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		// If token is already invalid/expired, nothing to revoke
		return nil
	}

	tokenHash := s.jwtService.HashToken(tokenString)
	expiresAt := time.Now().UTC().Add(s.jwtService.GetExpirationDuration())
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return s.revokedTokenRepo.Revoke(ctx, tokenHash, expiresAt)
}

// GetCurrentUser returns the sanitized user profile.
func (s *authService) GetCurrentUser(ctx context.Context, userID string) (*model.UserResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, repository.ErrUserNotFound
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := user.ToResponse()
	return &resp, nil
}
