package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

type pdfTextExtractor struct{}

// NewPDFTextExtractor creates an extractor for PDF files.
func NewPDFTextExtractor() TextExtractor {
	return &pdfTextExtractor{}
}

// ExtractText extracts plain text from a PDF reader.
// It guards against malformed PDF panics and handles multi-page documents.
func (e *pdfTextExtractor) ExtractText(ctx context.Context, reader io.ReaderAt, size int64) (outText string, retErr error) {
	if reader == nil || size <= 0 {
		return "", ErrInvalidPDF
	}

	// Guard against internal panics in third-party PDF parser
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("%w: parser error: %v", ErrInvalidPDF, r)
		}
	}()

	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPDF, err)
	}

	numPages := pdfReader.NumPage()
	if numPages <= 0 {
		return "", ErrExtractionFailed
	}

	var textBuilder strings.Builder

	// Attempt bulk plain text extraction first
	plainTextReader, err := pdfReader.GetPlainText()
	if err == nil && plainTextReader != nil {
		var buf bytes.Buffer
		if _, readErr := buf.ReadFrom(plainTextReader); readErr == nil && buf.Len() > 0 {
			textBuilder.Write(buf.Bytes())
		}
	}

	// If bulk extraction yielded no text, fallback to page-by-page extraction
	if strings.TrimSpace(textBuilder.String()) == "" {
		for pageIndex := 1; pageIndex <= numPages; pageIndex++ {
			p := pdfReader.Page(pageIndex)
			if p.V.IsNull() {
				continue
			}
			pageText, pErr := p.GetPlainText(nil)
			if pErr == nil && pageText != "" {
				textBuilder.WriteString(pageText)
				textBuilder.WriteString("\n")
			}
		}
	}

	extracted := strings.TrimSpace(textBuilder.String())
	if extracted == "" {
		return "", ErrExtractionFailed
	}

	if len(extracted) > MaxExtractedTextLength {
		return "", ErrCVTextTooLarge
	}

	return extracted, nil
}
