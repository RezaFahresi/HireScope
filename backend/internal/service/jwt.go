package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"hirescope/backend/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrExpiredToken          = errors.New("token has expired")
	ErrUnexpectedSigningAlgo = errors.New("unexpected signing algorithm")
	ErrMalformedToken        = errors.New("malformed token")
)

// Claims represents JWT claims for HireScope.
type Claims struct {
	Email string     `json:"email"`
	Role  model.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTService defines methods for creating and validating real cryptographically signed JWTs.
type JWTService interface {
	GenerateToken(user *model.User) (tokenString string, expiresInSeconds int64, err error)
	ValidateToken(tokenString string) (*Claims, error)
	HashToken(tokenString string) string
	GetExpirationDuration() time.Duration
}

type jwtService struct {
	secret         []byte
	expirationTime time.Duration
}

// NewJWTService creates a new instance of JWTService using the specified secret and expiration hours.
func NewJWTService(secret string, expirationHours int) (JWTService, error) {
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}
	if expirationHours <= 0 {
		expirationHours = 24
	}
	return &jwtService{
		secret:         []byte(secret),
		expirationTime: time.Duration(expirationHours) * time.Hour,
	}, nil
}

// GenerateToken generates a real HS256-signed JWT token using github.com/golang-jwt/jwt/v5.
func (s *jwtService) GenerateToken(user *model.User) (string, int64, error) {
	if user == nil {
		return "", 0, errors.New("user cannot be nil")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.expirationTime)

	claims := Claims{
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hirescope",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign JWT: %w", err)
	}

	expiresInSeconds := int64(s.expirationTime.Seconds())
	return tokenString, expiresInSeconds, nil
}

// ValidateToken validates a real JWT string using github.com/golang-jwt/jwt/v5.
// It verifies the cryptographic signature, checks the expiration, and enforces the HS256 algorithm.
func (s *jwtService) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMalformedToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Explicitly verify the signing algorithm is HMAC / HS256. Reject alg:none and asymmetric algs.
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("%w: expected %s, got %v", ErrUnexpectedSigningAlgo, jwt.SigningMethodHS256.Alg(), token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrMalformedToken
		}
		if errors.Is(err, ErrUnexpectedSigningAlgo) {
			return nil, ErrUnexpectedSigningAlgo
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Subject == "" || claims.Email == "" || !claims.Role.IsValid() {
		return nil, fmt.Errorf("%w: missing required claims", ErrInvalidToken)
	}

	return claims, nil
}

// HashToken calculates SHA-256 hash of the token string for safe revocation storage.
func (s *jwtService) HashToken(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return hex.EncodeToString(hash[:])
}

// GetExpirationDuration returns the configured token validity duration.
func (s *jwtService) GetExpirationDuration() time.Duration {
	return s.expirationTime
}
