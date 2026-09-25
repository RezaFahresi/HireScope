package migrations

import (
	"fmt"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// RunMigrations executes non-destructive schema migrations for the database.
func RunMigrations(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.User{},
		&model.RevokedToken{},
		&model.Job{},
		&model.JobRequirement{},
		&model.Candidate{},
		&model.CandidateDocument{},
		&model.CandidateEducation{},
		&model.CandidateExperience{},
		&model.CandidateSkill{},
		&model.JobCandidate{},
		&model.ScreeningResult{},
		&model.ScreeningMatch{},
		&model.CandidateNote{},
		&model.AuditLog{},
		&model.Interview{},
		&model.InterviewInterviewer{},
		&model.InterviewEmailDelivery{},
		&model.CalendarConnection{},
		&model.CalendarEvent{},
	)
	if err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	return nil
}
