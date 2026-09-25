package config

import (
	"testing"
)

func TestConfigValidation_MissingRequiredVars(t *testing.T) {
	cfg := &Config{
		AppEnv:  "development",
		AppPort: "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing configuration, got nil")
	}
}

func TestConfigValidation_MissingJWTSecret(t *testing.T) {
	cfg := &Config{
		AppEnv:     "development",
		AppPort:    "8080",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "hirescope",
		DBUser:     "postgres",
		DBPassword: "secretpassword",
		DBSSLMode:  "disable",
		JWTSecret:  "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing JWT_SECRET, got nil")
	}
}

func TestConfigValidation_Success(t *testing.T) {
	cfg := &Config{
		AppEnv:             "development",
		AppPort:            "8080",
		DBHost:             "localhost",
		DBPort:             "5432",
		DBName:             "hirescope",
		DBUser:             "postgres",
		DBPassword:         "secretpassword",
		DBSSLMode:          "disable",
		JWTSecret:          "very-secure-jwt-secret-key-12345",
		JWTExpirationHours: 24,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no validation error, got: %v", err)
	}

	dsn := cfg.DSN()
	expectedDSN := "host=localhost port=5432 user=postgres password=secretpassword dbname=hirescope sslmode=disable"
	if dsn != expectedDSN {
		t.Fatalf("Expected DSN '%s', got '%s'", expectedDSN, dsn)
	}
}
