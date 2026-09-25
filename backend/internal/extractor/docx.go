package extractor

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

type docxTextExtractor struct{}

// NewDOCXTextExtractor creates an extractor for DOCX files.
func NewDOCXTextExtractor() TextExtractor {
	return &docxTextExtractor{}
}

// ExtractText safely extracts text from a DOCX OpenXML archive.
// It explicitly ignores any macros, scripts, external references, or executable content.
func (e *docxTextExtractor) ExtractText(ctx context.Context, reader io.ReaderAt, size int64) (outText string, retErr error) {
	if reader == nil || size <= 0 {
		return "", ErrInvalidDOCX
	}

	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("%w: parser error: %v", ErrInvalidDOCX, r)
		}
	}()

	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return "", fmt.Errorf("%w: not a valid zip archive: %v", ErrInvalidDOCX, err)
	}

	// 1. Verify that this is genuinely a DOCX archive and not an arbitrary zip file
	var docFile *zip.File
	var contentTypesFound bool
	var headerFiles []*zip.File

	for _, file := range zipReader.File {
		if file.Name == "[Content_Types].xml" {
			contentTypesFound = true
		}
		if file.Name == "word/document.xml" {
			docFile = file
		}
		if strings.HasPrefix(file.Name, "word/header") && strings.HasSuffix(file.Name, ".xml") {
			headerFiles = append(headerFiles, file)
		}
	}

	if !contentTypesFound || docFile == nil {
		return "", fmt.Errorf("%w: missing WordprocessingML document structure", ErrInvalidDOCX)
	}

	var textBuilder strings.Builder

	// Extract headers first if any (e.g. contact info often placed in header)
	for _, hf := range headerFiles {
		hText, _ := extractXMLText(hf)
		if hText != "" {
			textBuilder.WriteString(hText)
			textBuilder.WriteString("\n")
		}
	}

	// Extract main body
	bodyText, err := extractXMLText(docFile)
	if err != nil {
		return "", fmt.Errorf("%w: failed to read document.xml: %v", ErrInvalidDOCX, err)
	}
	textBuilder.WriteString(bodyText)

	extracted := strings.TrimSpace(textBuilder.String())
	if extracted == "" {
		return "", ErrExtractionFailed
	}

	if len(extracted) > MaxExtractedTextLength {
		return "", ErrCVTextTooLarge
	}

	return extracted, nil
}

// extractXMLText decodes OpenXML paragraph and table text streams.
func extractXMLText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var sb strings.Builder
	var inText bool

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "p":
				// Start of a paragraph
			case "t":
				inText = true
			case "tab":
				sb.WriteString("\t")
			case "br", "cr":
				sb.WriteString("\n")
			}
		case xml.EndElement:
			switch elem.Name.Local {
			case "p":
				sb.WriteString("\n")
			case "t":
				inText = false
			case "tc":
				sb.WriteString("\t")
			case "tr":
				sb.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				sb.Write(elem)
			}
		}
	}

	return sb.String(), nil
}
