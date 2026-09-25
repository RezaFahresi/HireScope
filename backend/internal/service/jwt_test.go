package service

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"hirescope/backend/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTService_RealTokenGenerationAndValidation(t *testing.T) {
	secret := "test-super-secret-key-32-bytes-long!!"
	svc, err := NewJWTService(secret, 1)
	if err != nil {
		t.Fatalf("Failed to initialize JWT service: %v", err)
	}

	user := &model.User{
		ID:    "user-uuid-12345",
		Name:  "Jane Recruiter",
		Email: "jane@hirescope.local",
		Role:  model.RoleRecruiter,
	}

	// 1. Generate real token
	tokenString, expiresIn, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if expiresIn != 3600 {
		t.Fatalf("Expected expires in 3600, got %d", expiresIn)
	}

	// Token must have 3 segments separated by dots
	segments := strings.Split(tokenString, ".")
	if len(segments) != 3 {
		t.Fatalf("Expected 3 segments in JWT, got %d", len(segments))
	}

	// 2. Validate token
	claims, err := svc.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.Subject != user.ID {
		t.Fatalf("Expected subject %s, got %s", user.ID, claims.Subject)
	}
	if claims.Email != user.Email {
		t.Fatalf("Expected email %s, got %s", user.Email, claims.Email)
	}
	if claims.Role != model.RoleRecruiter {
		t.Fatalf("Expected role RECRUITER, got %s", claims.Role)
	}
	if claims.Issuer != "hirescope" {
		t.Fatalf("Expected issuer 'hirescope', got %s", claims.Issuer)
	}
}

func TestJWTService_AdminRoleClaims(t *testing.T) {
	svc, _ := NewJWTService("admin-secret-key-1234567890123456", 2)
	adminUser := &model.User{
		ID:    "admin-uuid-9999",
		Name:  "System Administrator",
		Email: "admin@hirescope.local",
		Role:  model.RoleAdmin,
	}

	token, _, err := svc.GenerateToken(adminUser)
	if err != nil {
		t.Fatalf("Failed to generate admin token: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate admin token: %v", err)
	}

	if claims.Role != model.RoleAdmin {
		t.Fatalf("Expected role ADMIN, got %s", claims.Role)
	}
}

func TestJWTService_RejectsWrongSecret(t *testing.T) {
	svc1, _ := NewJWTService("secret-one-12345678901234567890", 1)
	svc2, _ := NewJWTService("secret-two-12345678901234567890", 1)

	user := &model.User{
		ID:    "u-1",
		Email: "user@hirescope.local",
		Role:  model.RoleRecruiter,
	}

	token, _, err := svc1.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate with different secret must fail
	_, err = svc2.ValidateToken(token)
	if err == nil {
		t.Fatal("Expected validation error for wrong secret, got nil")
	}
}

func TestJWTService_RejectsExpiredToken(t *testing.T) {
	secret := []byte("secret-key-for-expired-token-test")
	// Create token that was already expired 1 hour ago
	expiredClaims := Claims{
		Email: "expired@hirescope.local",
		Role:  model.RoleRecruiter,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-expired",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    "hirescope",
		},
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := rawToken.SignedString(secret)
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	svc, _ := NewJWTService(string(secret), 1)
	_, err = svc.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("Expected error for expired token, got nil")
	}
}

func TestJWTService_RejectsMalformedToken(t *testing.T) {
	svc, _ := NewJWTService("secret-key-malformed-test-123456", 1)

	testCases := []string{
		"",
		"not-a-jwt",
		"header.payload",
		"a.b.c.d",
		"invalid.base64!!.token",
	}

	for _, tc := range testCases {
		_, err := svc.ValidateToken(tc)
		if err == nil {
			t.Fatalf("Expected error for malformed token '%s', got nil", tc)
		}
	}
}

func TestJWTService_RejectsAlgNone(t *testing.T) {
	// Construct an unsigned token with alg: none
	claims := Claims{
		Email: "hacker@test.com",
		Role:  model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "hacker-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("Failed to create none token: %v", err)
	}

	svc, _ := NewJWTService("secret-key-1234567890123456", 1)
	_, err = svc.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("Expected validation error for alg: none, got nil")
	}
}

func TestJWTService_RejectsUnexpectedAlgorithmRS256(t *testing.T) {
	// Create RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	claims := Claims{
		Email: "rsa@hirescope.local",
		Role:  model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "rsa-subject",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	rsaToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := rsaToken.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign RSA token: %v", err)
	}

	svc, _ := NewJWTService("hmac-secret-1234567890123456", 1)
	_, err = svc.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("Expected validation to reject RS256 token when expecting HS256, got nil")
	}
}

func TestJWTService_TokenHashing(t *testing.T) {
	svc, _ := NewJWTService("secret-hashing-1234567890123456", 1)
	token := "sample.token.string"
	hash1 := svc.HashToken(token)
	hash2 := svc.HashToken(token)

	if hash1 != hash2 {
		t.Fatal("Token hash must be deterministic")
	}
	if len(hash1) != 64 { // SHA-256 hex string has 64 characters
		t.Fatalf("Expected 64 hex characters for SHA-256 hash, got %d", len(hash1))
	}
}
