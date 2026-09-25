package extractor

import (
	"context"
	"errors"
	"io"
)

const (
	// MaxExtractedTextLength is 5 MB of UTF-8 text.
	MaxExtractedTextLength = 5000000
)

var (
	ErrExtractionFailed  = errors.New("no extractable text was found in the document")
	ErrUnsupportedFormat = errors.New("unsupported document format for text extraction")
	ErrInvalidDocument   = errors.New("the uploaded document is not a valid file")
	ErrInvalidPDF        = errors.New("the uploaded document is not a valid PDF file")
	ErrInvalidDOCX       = errors.New("the uploaded document is not a valid DOCX file")
	ErrCVTextTooLarge    = errors.New("extracted CV text exceeds maximum allowed size (5 MB)")
)

// TextExtractor defines operations to extract text from a binary document.
type TextExtractor interface {
	ExtractText(ctx context.Context, reader io.ReaderAt, size int64) (string, error)
}
