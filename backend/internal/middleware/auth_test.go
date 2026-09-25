package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// memoryRevokedTokenRepo implements repository.RevokedTokenRepository in-memory for safe testing.
type memoryRevokedTokenRepo struct {
	revoked map[string]time.Time
}

func newMemoryRevokedTokenRepo() *memoryRevokedTokenRepo {
	return &memoryRevokedTokenRepo{revoked: make(map[string]time.Time)}
}

func (r *memoryRevokedTokenRepo) Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	r.revoked[tokenHash] = expiresAt
	return nil
}

func (r *memoryRevokedTokenRepo) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	exp, exists := r.revoked[tokenHash]
	if !exists {
		return false, nil
	}
	return exp.After(time.Now().UTC()), nil
}

func (r *memoryRevokedTokenRepo) CleanExpired(ctx context.Context) error {
	now := time.Now().UTC()
	for h, exp := range r.revoked {
		if exp.Before(now) {
			delete(r.revoked, h)
		}
	}
	return nil
}

func setupTestRouter(jwtService service.JWTService, tokenRepo *memoryRevokedTokenRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	authGroup := router.Group("/test")
	authGroup.Use(AuthMiddleware(jwtService, tokenRepo))
	{
		authGroup.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"userID":   c.GetString(ContextUserID),
				"userRole": c.GetString(ContextUserRole),
			})
		})

		authGroup.GET("/admin-only", RequireRole(model.RoleAdmin), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "admin_granted"})
		})

		authGroup.GET("/recruiter-only", RequireRole(model.RoleRecruiter), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "recruiter_granted"})
		})
	}

	return router
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for missing token, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
	req.Header.Set("Authorization", "Token abcdef12345")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for invalid Bearer format, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for invalid token, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	user := &model.User{
		ID:    "user-12345",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	token, _, err := jwtSvc.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for valid token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_RevokedTokenRejected(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	user := &model.User{
		ID:    "user-revoked",
		Email: "revoked@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	token, _, _ := jwtSvc.GenerateToken(user)

	// Revoke token
	hash := jwtSvc.HashToken(token)
	_ = repo.Revoke(context.Background(), hash, time.Now().Add(1*time.Hour))

	req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 for revoked token, got %d", w.Code)
	}
}

func TestRoleMiddleware_AdminOnly(t *testing.T) {
	jwtSvc, _ := service.NewJWTService("secret-for-auth-middleware-tests!!", 1)
	repo := newMemoryRevokedTokenRepo()
	router := setupTestRouter(jwtSvc, repo)

	adminUser := &model.User{ID: "adm-1", Email: "admin@hirescope.local", Role: model.RoleAdmin}
	adminToken, _, _ := jwtSvc.GenerateToken(adminUser)

	recruiterUser := &model.User{ID: "rec-1", Email: "rec@hirescope.local", Role: model.RoleRecruiter}
	recruiterToken, _, _ := jwtSvc.GenerateToken(recruiterUser)

	// 1. Admin accessing admin-only route -> 200 OK
	req1, _ := http.NewRequest(http.MethodGet, "/test/admin-only", nil)
	req1.Header.Set("Authorization", "Bearer "+adminToken)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("Expected 200 for admin on admin-only route, got %d", w1.Code)
	}

	// 2. Recruiter accessing admin-only route -> 403 Forbidden
	req2, _ := http.NewRequest(http.MethodGet, "/test/admin-only", nil)
	req2.Header.Set("Authorization", "Bearer "+recruiterToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 for recruiter on admin-only route, got %d", w2.Code)
	}

	// 3. Recruiter accessing recruiter-only route -> 200 OK
	req3, _ := http.NewRequest(http.MethodGet, "/test/recruiter-only", nil)
	req3.Header.Set("Authorization", "Bearer "+recruiterToken)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("Expected 200 for recruiter on recruiter-only route, got %d", w3.Code)
	}
}
