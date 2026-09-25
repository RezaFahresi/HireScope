package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"

	"gorm.io/gorm"
)

// SeedDevelopmentUsers creates default Admin and Recruiter users if the users table is empty.
func SeedDevelopmentUsers(db *gorm.DB) error {
	userRepo := repository.NewUserRepository(db)
	ctx := context.Background()

	count, err := userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		// Users already exist, skip seeding to avoid touching existing data
		return nil
	}

	log.Println("Seeding initial development users...")

	// 1. Admin user
	admin := &model.User{
		Name:  "Admin HireScope",
		Email: "admin@hirescope.local",
		Role:  model.RoleAdmin,
	}
	if err := admin.SetPassword("AdminSecure2026!"); err != nil {
		return fmt.Errorf("failed to set admin password: %w", err)
	}
	if err := userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}
	log.Printf("Seeded default admin user: %s (Role: %s)", admin.Email, admin.Role)

	// 2. Recruiter user
	recruiter := &model.User{
		Name:  "Sarah Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	if err := recruiter.SetPassword("RecruiterSecure2026!"); err != nil {
		return fmt.Errorf("failed to set recruiter password: %w", err)
	}
	if err := userRepo.Create(ctx, recruiter); err != nil {
		return fmt.Errorf("failed to seed recruiter user: %w", err)
	}
	log.Printf("Seeded default recruiter user: %s (Role: %s)", recruiter.Email, recruiter.Role)

	return nil
}

// SeedDevelopmentJobs creates a minimal example job vacancy with requirements if the jobs table is empty.
func SeedDevelopmentJobs(db *gorm.DB) error {
	userRepo := repository.NewUserRepository(db)
	jobRepo := repository.NewJobRepository(db)
	ctx := context.Background()

	var jobCount int64
	if err := db.WithContext(ctx).Model(&model.Job{}).Count(&jobCount).Error; err != nil {
		return fmt.Errorf("failed to count jobs: %w", err)
	}

	if jobCount > 0 {
		return nil
	}

	// Find recruiter user to set as creator
	recruiter, err := userRepo.FindByEmail(ctx, "recruiter@hirescope.local")
	if err != nil {
		return nil // If user doesn't exist yet, skip
	}

	closingDate := time.Now().AddDate(0, 3, 0)
	sampleJob := &model.Job{
		Code:           "JOB-2026-0001",
		Title:          "IT Business Analyst",
		Description:    "Responsible for gathering business requirements, creating BPMN diagrams, and bridging the gap between business stakeholders and software engineering teams.",
		Department:     "Information Technology",
		Location:       "Jakarta",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		ClosingDate:    &closingDate,
		CreatedBy:      recruiter.ID,
	}

	if err := jobRepo.Create(ctx, sampleJob); err != nil {
		return fmt.Errorf("failed to seed sample job: %w", err)
	}

	sampleRequirements := []model.JobRequirement{
		{
			JobID:       sampleJob.ID,
			Category:    model.CategorySkill,
			Requirement: "Business Process Model and Notation (BPMN 2.0)",
			Importance:  model.ImportanceRequired,
		},
		{
			JobID:       sampleJob.ID,
			Category:    model.CategorySkill,
			Requirement: "Business Requirement Document (BRD) & Functional Requirement Document (FRD) writing",
			Importance:  model.ImportanceRequired,
		},
		{
			JobID:       sampleJob.ID,
			Category:    model.CategoryExperience,
			Requirement: "At least 3 years experience as a Business Analyst in finance or tech",
			Importance:  model.ImportanceRequired,
		},
		{
			JobID:       sampleJob.ID,
			Category:    model.CategoryEducation,
			Requirement: "Bachelor's degree in Computer Science, Information Systems, or related field",
			Importance:  model.ImportanceRequired,
		},
		{
			JobID:       sampleJob.ID,
			Category:    model.CategorySkill,
			Requirement: "SQL querying and relational database analysis",
			Importance:  model.ImportancePreferred,
		},
	}

	for i := range sampleRequirements {
		if err := jobRepo.CreateRequirement(ctx, &sampleRequirements[i]); err != nil {
			return fmt.Errorf("failed to seed sample requirement: %w", err)
		}
	}

	log.Printf("Seeded sample job: %s (%s) with %d requirements", sampleJob.Title, sampleJob.Code, len(sampleRequirements))
	return nil
}
