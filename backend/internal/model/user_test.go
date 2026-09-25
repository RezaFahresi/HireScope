package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUser_PasswordSecurity(t *testing.T) {
	u := &User{
		Name:  "Test Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  RoleRecruiter,
	}

	// 1. Password must be >= 8 chars
	err := u.SetPassword("short")
	if err == nil {
		t.Fatal("Expected error for password shorter than 8 characters")
	}

	// 2. Set valid password
	err = u.SetPassword("StrongPassword123!")
	if err != nil {
		t.Fatalf("Expected no error setting valid password, got: %v", err)
	}

	// 3. PasswordHash must not be empty and must not be plaintext
	if u.PasswordHash == "" {
		t.Fatal("PasswordHash should not be empty")
	}
	if u.PasswordHash == "StrongPassword123!" {
		t.Fatal("PasswordHash must NEVER be plaintext")
	}

	// 4. Verify password
	if !u.CheckPassword("StrongPassword123!") {
		t.Fatal("Expected CheckPassword to succeed with correct password")
	}

	// 5. Wrong password must fail
	if u.CheckPassword("WrongPassword123!") {
		t.Fatal("Expected CheckPassword to fail with incorrect password")
	}

	if u.CheckPassword("") {
		t.Fatal("Expected CheckPassword to fail with empty password")
	}
}

func TestUser_PasswordHashNeverExposedInJSON(t *testing.T) {
	u := &User{
		ID:           "test-uuid-1234",
		Name:         "Admin User",
		Email:        "admin@hirescope.local",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz123456",
		Role:         RoleAdmin,
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "password_hash") || strings.Contains(jsonStr, "PasswordHash") || strings.Contains(jsonStr, "abcdefghijkl") {
		t.Fatalf("Password hash was leaked in User JSON serialization: %s", jsonStr)
	}

	resp := u.ToResponse()
	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal UserResponse: %v", err)
	}
	respJsonStr := string(respData)
	if strings.Contains(respJsonStr, "password") {
		t.Fatalf("Password appeared in UserResponse: %s", respJsonStr)
	}
}

func TestUser_EmailNormalization(t *testing.T) {
	raw := "  Admin.Tester@HireScope.LOCAL  "
	normalized := NormalizeEmail(raw)
	expected := "admin.tester@hirescope.local"
	if normalized != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, normalized)
	}
}
