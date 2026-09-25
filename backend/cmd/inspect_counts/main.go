package main

import (
	"fmt"
	"log"

	"hirescope/backend/internal/config"
	"hirescope/backend/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbWrapper, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	tables := []string{
		"users",
		"jobs",
		"job_requirements",
		"candidates",
		"job_candidates",
		"candidate_educations",
		"candidate_experiences",
		"candidate_skills",
		"candidate_documents",
		"screening_results",
		"screening_matches",
		"candidate_notes",
		"interviews",
		"interview_interviewers",
		"interview_email_deliveries",
		"calendar_connections",
		"calendar_events",
		"audit_logs",
		"revoked_tokens",
	}

	fmt.Println("=== DATABASE ROW COUNTS (BASELINE) ===")
	for _, t := range tables {
		var count int64
		if err := dbWrapper.DB.Table(t).Count(&count).Error; err != nil {
			fmt.Printf("%-28s: ERROR (%v)\n", t, err)
		} else {
			fmt.Printf("%-28s: %d\n", t, count)
		}
	}
}
