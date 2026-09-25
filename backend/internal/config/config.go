package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all backend application configuration.
type Config struct {
	AppEnv             string
	AppPort            string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBSSLMode          string
	JWTSecret          string
	JWTExpirationHours     int
	CVStorageDir           string
	EmailProvider          string
	EmailEnabled           bool
	ResendAPIKey           string
	EmailFromName          string
	EmailFromAddress       string
	EmailReplyTo           string
	EmailBaseURL           string
	InterviewReminderHours int
	GoogleClientID              string
	GoogleClientSecret          string
	GoogleRedirectURL           string
	MicrosoftClientID           string
	MicrosoftClientSecret       string
	MicrosoftTenantID           string
	MicrosoftRedirectURL        string
	CalendarTokenEncryptionKey  string
}

// Load loads configuration from environment variables and an optional .env file.
// It enforces strict validation on required environment variables.
func Load() (*Config, error) {
	// Attempt to load .env file if it exists, ignore if not present
	_ = godotenv.Load()

	expHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil || expHours <= 0 {
		expHours = 24
	}

	reminderHours, err := strconv.Atoi(getEnv("INTERVIEW_REMINDER_HOURS", "24"))
	if err != nil || reminderHours <= 0 {
		reminderHours = 24
	}

	emailEnabled := false
	if enabledVal := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_ENABLED"))); enabledVal == "true" || enabledVal == "1" || enabledVal == "yes" {
		emailEnabled = true
	}

	cfg := &Config{
		AppEnv:                     getEnv("APP_ENV", "development"),
		AppPort:                    getEnv("APP_PORT", "8080"),
		DBHost:                     os.Getenv("DB_HOST"),
		DBPort:                     os.Getenv("DB_PORT"),
		DBName:                     os.Getenv("DB_NAME"),
		DBUser:                     os.Getenv("DB_USER"),
		DBPassword:                 os.Getenv("DB_PASSWORD"),
		DBSSLMode:                  getEnv("DB_SSLMODE", "disable"),
		JWTSecret:                  os.Getenv("JWT_SECRET"),
		JWTExpirationHours:         expHours,
		CVStorageDir:               getEnv("CV_STORAGE_DIR", "./storage/cv_documents"),
		EmailProvider:              getEnv("EMAIL_PROVIDER", "resend"),
		EmailEnabled:               emailEnabled,
		ResendAPIKey:               os.Getenv("RESEND_API_KEY"),
		EmailFromName:              getEnv("EMAIL_FROM_NAME", "HireScope"),
		EmailFromAddress:           os.Getenv("EMAIL_FROM_ADDRESS"),
		EmailReplyTo:               os.Getenv("EMAIL_REPLY_TO"),
		EmailBaseURL:               getEnv("EMAIL_BASE_URL", "http://localhost:5173"),
		InterviewReminderHours:     reminderHours,
		GoogleClientID:             os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:         os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:          getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/calendar/google/callback"),
		MicrosoftClientID:          os.Getenv("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret:      os.Getenv("MICROSOFT_CLIENT_SECRET"),
		MicrosoftTenantID:          getEnv("MICROSOFT_TENANT_ID", "common"),
		MicrosoftRedirectURL:       getEnv("MICROSOFT_REDIRECT_URL", "http://localhost:8080/api/v1/calendar/microsoft/callback"),
		CalendarTokenEncryptionKey: os.Getenv("CALENDAR_TOKEN_ENCRYPTION_KEY"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate verifies that all mandatory configuration parameters are provided.
func (c *Config) Validate() error {
	var missing []string

	if strings.TrimSpace(c.DBHost) == "" {
		missing = append(missing, "DB_HOST")
	}
	if strings.TrimSpace(c.DBPort) == "" {
		missing = append(missing, "DB_PORT")
	}
	if strings.TrimSpace(c.DBName) == "" {
		missing = append(missing, "DB_NAME")
	}
	if strings.TrimSpace(c.DBUser) == "" {
		missing = append(missing, "DB_USER")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		missing = append(missing, "JWT_SECRET")
	}

	// When email is explicitly enabled, validate required provider settings
	if c.EmailEnabled {
		if c.EmailProvider == "resend" && strings.TrimSpace(c.ResendAPIKey) == "" {
			missing = append(missing, "RESEND_API_KEY")
		}
		if strings.TrimSpace(c.EmailFromAddress) == "" {
			missing = append(missing, "EMAIL_FROM_ADDRESS")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// DSN returns the PostgreSQL connection string formatted for GORM.
func (c *Config) DSN() string {
	if c.DBPassword == "" {
		return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			c.DBHost, c.DBPort, c.DBUser, c.DBName, c.DBSSLMode)
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
