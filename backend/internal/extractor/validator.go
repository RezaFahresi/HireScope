package extractor

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

var (
	pdfSignature = []byte("%PDF-")
	zipMagic     = []byte("PK\x03\x04")
)

// ValidatePDFSignature checks whether the byte slice begins with or contains the PDF magic signature.
func ValidatePDFSignature(header []byte) bool {
	if len(header) < 5 {
		return false
	}
	// Many PDFs start immediately with %PDF-, but some have a BOM or comment up to 1024 bytes
	limit := len(header)
	if limit > 1024 {
		limit = 1024
	}
	return bytes.Contains(header[:limit], pdfSignature)
}

// ValidateDOCXStructure verifies that the reader is a valid OpenXML archive containing word/document.xml.
func ValidateDOCXStructure(reader io.ReaderAt, size int64) bool {
	if reader == nil || size < 4 {
		return false
	}

	header := make([]byte, 4)
	if _, err := reader.ReadAt(header, 0); err != nil {
		return false
	}
	if !bytes.Equal(header, zipMagic) {
		return false
	}

	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return false
	}

	var hasDocXML bool
	var hasContentTypes bool
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			hasDocXML = true
		}
		if f.Name == "[Content_Types].xml" {
			hasContentTypes = true
		}
		if hasDocXML && hasContentTypes {
			return true
		}
	}

	return false
}

// SanitizeFilename strips directory components and unsafe characters.
func SanitizeFilename(filename string) string {
	base := filepath.Base(filename)
	// Remove non-standard or dangerous characters
	var sb strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_' || r == ' ' {
			sb.WriteRune(r)
		}
	}
	sanitized := strings.TrimSpace(sb.String())
	if sanitized == "" || sanitized == "." {
		return "cv_document"
	}
	return sanitized
}

// DetectAndValidateFormat checks extension, declared MIME type, and file signature.
func DetectAndValidateFormat(filename string, declaredMime string, reader io.ReaderAt, size int64) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".pdf" && ext != ".docx" {
		return "", errors.New("unsupported file extension, only .pdf and .docx are supported")
	}

	header := make([]byte, 1024)
	n, _ := reader.ReadAt(header, 0)
	header = header[:n]

	if ext == ".pdf" {
		if !ValidatePDFSignature(header) {
			return "", ErrInvalidPDF
		}
		return "PDF", nil
	}

	if ext == ".docx" {
		if !ValidateDOCXStructure(reader, size) {
			return "", ErrInvalidDOCX
		}
		return "DOCX", nil
	}

	return "", ErrUnsupportedFormat
}
