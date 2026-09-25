package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Role represents user authorization roles in HireScope.
type Role string

const (
	RoleAdmin     Role = "ADMIN"
	RoleRecruiter Role = "RECRUITER"
)

// IsValid checks if the role is a recognized role.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleRecruiter
}

// User represents a system user in HireScope.
type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         Role      `gorm:"type:varchar(50);not null;index" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate generates a UUID for the user and normalizes the email.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	u.Email = NormalizeEmail(u.Email)
	return nil
}

// BeforeUpdate normalizes email before updating.
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.Email = NormalizeEmail(u.Email)
	return nil
}

// SetPassword hashes and sets the user's password using bcrypt.
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the provided plain password against the stored bcrypt hash.
func (u *User) CheckPassword(password string) bool {
	if u.PasswordHash == "" || password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// UserResponse represents safe user data returned in API responses.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a User entity into a safe UserResponse (without password hash).
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// NormalizeEmail converts email to lowercase and trims whitespace.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
