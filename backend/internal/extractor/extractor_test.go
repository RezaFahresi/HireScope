package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func createTestDocx(content string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// [Content_Types].xml
	ct, _ := zw.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
</Types>`))

	// word/document.xml
	doc, _ := zw.Create("word/document.xml")
	_, _ = doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>` + content + `</w:t></w:r></w:p>
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Table Cell 1</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Table Cell 2</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`))

	_ = zw.Close()
	return buf.Bytes()
}

func createNonDocxZip() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("random.txt")
	_, _ = f.Write([]byte("not a word document"))
	_ = zw.Close()
	return buf.Bytes()
}

func buildValidPDF(content string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)

	// obj 1: Catalog
	offsets[1] = b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// obj 2: Pages
	offsets[2] = b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// obj 3: Page
	offsets[3] = b.Len()
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// obj 4: Stream
	streamContent := "BT /F1 12 Tf 72 712 Td (" + content + ") Tj ET\n"
	offsets[4] = b.Len()
	b.WriteString("4 0 obj\n<< /Length ")
	b.WriteString(fmt.Sprintf("%d", len(streamContent)))
	b.WriteString(" >>\nstream\n")
	b.WriteString(streamContent)
	b.WriteString("endstream\nendobj\n")

	// obj 5: Font
	offsets[5] = b.Len()
	b.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// xref
	xrefOffset := b.Len()
	b.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))

	return b.Bytes()
}

func TestPDFTextExtractor_ValidPDF(t *testing.T) {
	extractor := NewPDFTextExtractor()
	data := buildValidPDF("Hello HireScope PDF Candidate")
	reader := bytes.NewReader(data)

	text, err := extractor.ExtractText(context.Background(), reader, int64(len(data)))
	if err != nil {
		t.Fatalf("expected successful PDF extraction, got: %v", err)
	}

	if !strings.Contains(text, "Hello HireScope PDF Candidate") {
		t.Errorf("extracted text missing expected content: got %q", text)
	}
}

func TestPDFTextExtractor_MalformedPDF(t *testing.T) {
	extractor := NewPDFTextExtractor()
	data := []byte("%PDF-1.4 malformed corrupt bytes not a real pdf structure")
	reader := bytes.NewReader(data)

	_, err := extractor.ExtractText(context.Background(), reader, int64(len(data)))
	if err == nil {
		t.Fatal("expected error for malformed PDF, got nil")
	}
}

func TestDOCXTextExtractor_ValidDOCX(t *testing.T) {
	extractor := NewDOCXTextExtractor()
	data := createTestDocx("John Doe - Senior Engineer")
	reader := bytes.NewReader(data)

	text, err := extractor.ExtractText(context.Background(), reader, int64(len(data)))
	if err != nil {
		t.Fatalf("expected successful DOCX extraction, got: %v", err)
	}

	if !strings.Contains(text, "John Doe - Senior Engineer") {
		t.Errorf("extracted text missing expected paragraph: got %q", text)
	}
	if !strings.Contains(text, "Table Cell 1") || !strings.Contains(text, "Table Cell 2") {
		t.Errorf("extracted text missing expected table content: got %q", text)
	}
}

func TestDOCXTextExtractor_RejectNonDocxZip(t *testing.T) {
	extractor := NewDOCXTextExtractor()
	data := createNonDocxZip()
	reader := bytes.NewReader(data)

	_, err := extractor.ExtractText(context.Background(), reader, int64(len(data)))
	if err == nil {
		t.Fatal("expected error rejecting arbitrary zip without docx structure, got nil")
	}
}

func TestValidator_SignaturesAndFilename(t *testing.T) {
	// PDF signature check
	if !ValidatePDFSignature([]byte("%PDF-1.7 ...")) {
		t.Error("expected valid PDF signature")
	}
	if ValidatePDFSignature([]byte("Not a PDF file")) {
		t.Error("expected invalid for non-PDF")
	}

	// DOCX signature check
	validDocx := createTestDocx("Test")
	if !ValidateDOCXStructure(bytes.NewReader(validDocx), int64(len(validDocx))) {
		t.Error("expected valid DOCX structure")
	}
	nonDocx := createNonDocxZip()
	if ValidateDOCXStructure(bytes.NewReader(nonDocx), int64(len(nonDocx))) {
		t.Error("expected non-DOCX zip to fail structure validation")
	}

	// Filename sanitization
	safe := SanitizeFilename("../../secret/my_resume#1!.pdf")
	if strings.Contains(safe, "/") || strings.Contains(safe, "\\") || strings.Contains(safe, "..") {
		t.Errorf("sanitization failed to remove path components: %s", safe)
	}
}
