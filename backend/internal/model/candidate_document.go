package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DocumentSourceType specifies the input medium of a candidate CV.
type DocumentSourceType string

const (
	SourceTypeText DocumentSourceType = "TEXT"
	SourceTypePDF  DocumentSourceType = "PDF"
	SourceTypeDOCX DocumentSourceType = "DOCX"
)

// IsValid checks if the source type is recognized.
func (s DocumentSourceType) IsValid() bool {
	return s == SourceTypeText || s == SourceTypePDF || s == SourceTypeDOCX
}

// CandidateDocument represents a candidate CV source and its raw text content.
type CandidateDocument struct {
	ID               string             `gorm:"type:uuid;primaryKey" json:"id"`
	CandidateID      string             `gorm:"type:uuid;not null;index" json:"candidate_id"`
	SourceType       DocumentSourceType `gorm:"type:varchar(50);not null;index" json:"source_type"`
	OriginalFilename *string            `gorm:"type:varchar(255)" json:"original_filename,omitempty"`
	MimeType         *string            `gorm:"type:varchar(100)" json:"mime_type,omitempty"`
	FileSize         *int64             `json:"file_size,omitempty"`
	StoragePath      *string            `gorm:"type:varchar(500)" json:"storage_path,omitempty"`
	RawText          string             `gorm:"type:text;not null" json:"raw_text"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// BeforeCreate assigns a UUID and validates document attributes.
func (d *CandidateDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if !d.SourceType.IsValid() {
		return errors.New("invalid document source type")
	}
	d.RawText = strings.TrimSpace(d.RawText)
	if d.SourceType == SourceTypeText && d.RawText == "" {
		return errors.New("raw text cannot be blank for TEXT document source")
	}
	return nil
}
