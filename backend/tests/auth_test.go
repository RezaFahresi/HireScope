package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// In-memory mock repositories for completely isolated, database-safe testing
type memoryUserRepo struct {
	users map[string]*model.User
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{users: make(map[string]*model.User)}
}

func (r *memoryUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	u, exists := r.users[model.NormalizeEmail(email)]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}

func (r *memoryUserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (r *memoryUserRepo) Create(ctx context.Context, user *model.User) error {
	r.users[model.NormalizeEmail(user.Email)] = user
	return nil
}

func (r *memoryUserRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.users)), nil
}

func (r *memoryUserRepo) List(ctx context.Context) ([]model.User, error) {
	var list []model.User
	for _, u := range r.users {
		list = append(list, *u)
	}
	return list, nil
}

type memoryRevokedRepo struct {
	revoked map[string]time.Time
}

func newMemoryRevokedRepo() *memoryRevokedRepo {
	return &memoryRevokedRepo{revoked: make(map[string]time.Time)}
}

func (r *memoryRevokedRepo) Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	r.revoked[tokenHash] = expiresAt
	return nil
}

func (r *memoryRevokedRepo) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	exp, exists := r.revoked[tokenHash]
	if !exists {
		return false, nil
	}
	return exp.After(time.Now().UTC()), nil
}

func (r *memoryRevokedRepo) CleanExpired(ctx context.Context) error {
	return nil
}

func setupAuthRouter() (*gin.Engine, *memoryUserRepo, service.JWTService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSvc, _ := service.NewJWTService("super-secret-jwt-key-for-handler-tests!!", 1)
	userRepo := newMemoryUserRepo()
	revokedRepo := newMemoryRevokedRepo()
	authSvc := service.NewAuthService(userRepo, revokedRepo, jwtSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	v1 := router.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)

			protectedAuth := authGroup.Group("")
			protectedAuth.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
			{
				protectedAuth.POST("/logout", authHandler.Logout)
				protectedAuth.GET("/me", authHandler.Me)
			}
		}
	}

	return router, userRepo, jwtSvc
}

func TestAuthHandler_LoginSuccess(t *testing.T) {
	router, userRepo, _ := setupAuthRouter()

	// Seed user
	user := &model.User{
		ID:    "u-1",
		Name:  "Jane Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = user.SetPassword("SecurePassword123!")
	_ = userRepo.Create(context.Background(), user)

	// Send login request
	body, _ := json.Marshal(handler.LoginRequest{
		Email:    "recruiter@hirescope.local",
		Password: "SecurePassword123!",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data service.LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// 1. Verify real JWT is returned with 3 dot-separated segments
	token := resp.Data.AccessToken
	if token == "" {
		t.Fatal("Access token should not be empty")
	}
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		t.Fatalf("Expected 3 segments in JWT, got %d", len(segments))
	}

	// 2. Verify token type and expiration
	if resp.Data.TokenType != "Bearer" {
		t.Fatalf("Expected TokenType 'Bearer', got '%s'", resp.Data.TokenType)
	}
	if resp.Data.ExpiresIn <= 0 {
		t.Fatalf("Expected ExpiresIn > 0, got %d", resp.Data.ExpiresIn)
	}

	// 3. Verify user info and ensure NO password hash is exposed
	if resp.Data.User.Email != user.Email {
		t.Fatalf("Expected email '%s', got '%s'", user.Email, resp.Data.User.Email)
	}
	rawResp := w.Body.String()
	if strings.Contains(rawResp, "password_hash") || strings.Contains(rawResp, "PasswordHash") {
		t.Fatalf("Response leaked password hash: %s", rawResp)
	}
}

func TestAuthHandler_LoginInvalidPassword(t *testing.T) {
	router, userRepo, _ := setupAuthRouter()

	user := &model.User{
		ID:    "u-1",
		Name:  "Jane Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = user.SetPassword("SecurePassword123!")
	_ = userRepo.Create(context.Background(), user)

	body, _ := json.Marshal(handler.LoginRequest{
		Email:    "recruiter@hirescope.local",
		Password: "WrongPassword!",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for incorrect password, got %d", w.Code)
	}
}

func TestAuthHandler_LoginUnknownEmail(t *testing.T) {
	router, _, _ := setupAuthRouter()

	body, _ := json.Marshal(handler.LoginRequest{
		Email:    "unknown@hirescope.local",
		Password: "Password123!",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for unknown email, got %d", w.Code)
	}
}

func TestAuthHandler_LoginValidationErrors(t *testing.T) {
	router, _, _ := setupAuthRouter()

	// Missing password
	body, _ := json.Marshal(map[string]string{
		"email": "test@hirescope.local",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for missing password, got %d", w.Code)
	}
}

func TestAuthHandler_MeEndpoint(t *testing.T) {
	router, userRepo, jwtSvc := setupAuthRouter()

	user := &model.User{
		ID:    "u-me-test",
		Name:  "Test Me User",
		Email: "me@hirescope.local",
		Role:  model.RoleAdmin,
	}
	_ = user.SetPassword("SecurePassword123!")
	_ = userRepo.Create(context.Background(), user)

	token, _, _ := jwtSvc.GenerateToken(user)

	// 1. Calling /auth/me with valid Bearer token -> 200 OK
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.UserResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID != user.ID || resp.Data.Email != user.Email || resp.Data.Role != model.RoleAdmin {
		t.Fatalf("Incorrect user profile returned: %+v", resp.Data)
	}

	// 2. Calling without token -> 401
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	wUnauth := httptest.NewRecorder()
	router.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for unauthenticated request, got %d", wUnauth.Code)
	}
}

func TestAuthHandler_LogoutRevocation(t *testing.T) {
	router, userRepo, jwtSvc := setupAuthRouter()

	user := &model.User{
		ID:    "u-logout",
		Name:  "Logout Tester",
		Email: "logout@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = user.SetPassword("SecurePassword123!")
	_ = userRepo.Create(context.Background(), user)

	token, _, _ := jwtSvc.GenerateToken(user)

	// 1. Verify token works before logout
	req1, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("Expected 200 before logout, got %d", w1.Code)
	}

	// 2. Call logout
	reqLogout, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	wLogout := httptest.NewRecorder()
	router.ServeHTTP(wLogout, reqLogout)
	if wLogout.Code != http.StatusOK {
		t.Fatalf("Expected 200 for logout, got %d", wLogout.Code)
	}

	// 3. Calling /auth/me with the same token now must return 401 (TOKEN_REVOKED)
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 after logout revocation, got %d", w2.Code)
	}
}

type failingRevokedRepo struct {
	memoryRevokedRepo
}

func (r *failingRevokedRepo) Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	return errors.New("database connection refused")
}

func TestAuthHandler_LogoutRevocationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSvc, _ := service.NewJWTService("super-secret-jwt-key-for-handler-tests!!", 1)
	userRepo := newMemoryUserRepo()
	revokedRepo := &failingRevokedRepo{}
	authSvc := service.NewAuthService(userRepo, revokedRepo, jwtSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	user := &model.User{
		ID:    "u-logout-fail",
		Name:  "Fail Tester",
		Email: "faillogout@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = user.SetPassword("SecurePassword123!")
	_ = userRepo.Create(context.Background(), user)
	token, _, _ := jwtSvc.GenerateToken(user)

	v1 := router.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			protectedAuth := authGroup.Group("")
			protectedAuth.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
			{
				protectedAuth.POST("/logout", authHandler.Logout)
			}
		}
	}

	reqLogout, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	wLogout := httptest.NewRecorder()
	router.ServeHTTP(wLogout, reqLogout)

	if wLogout.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 when revocation fails, got %d", wLogout.Code)
	}
}
