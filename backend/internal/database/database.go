package database

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"hirescope/backend/internal/config"
)

// DBWrapper wraps the GORM DB instance and underlying SQL DB.
type DBWrapper struct {
	DB    *gorm.DB
	SQLDB *sql.DB
}

// Connect initializes and returns a GORM PostgreSQL database connection.
func Connect(cfg *config.Config) (*DBWrapper, error) {
	dsn := cfg.DSN()

	gormLogLevel := logger.Silent
	if cfg.AppEnv == "development" {
		gormLogLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Verify database connection is alive
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBWrapper{
		DB:    db,
		SQLDB: sqlDB,
	}, nil
}

// Close closes the underlying SQL database connection.
func (w *DBWrapper) Close() error {
	if w.SQLDB != nil {
		return w.SQLDB.Close()
	}
	return nil
}
