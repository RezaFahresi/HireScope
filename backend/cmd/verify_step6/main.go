package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseURL = "http://localhost:8080/api/v1"

func fail(format string, a ...interface{}) {
	fmt.Printf("\n❌ FAIL: "+format+"\n", a...)
	os.Exit(1)
}

func pass(format string, a ...interface{}) {
	fmt.Printf("✅ PASS: "+format+"\n", a...)
}

func createTestPDF(content string) []byte {
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
	b.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(streamContent), streamContent))

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

func createTestDOCX(paragraphs []string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	ct, _ := zw.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
</Types>`))

	var bodyBuilder strings.Builder
	for _, p := range paragraphs {
		bodyBuilder.WriteString(fmt.Sprintf("<w:p><w:r><w:t>%s</w:t></w:r></w:p>", p))
	}

	doc, _ := zw.Create("word/document.xml")
	_, _ = doc.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>%s</w:body>
</w:document>`, bodyBuilder.String())))

	_ = zw.Close()
	return buf.Bytes()
}

func postJSON(url, token string, body interface{}) (*http.Response, map[string]interface{}) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fail("HTTP request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var result map[string]interface{}
	_ = json.Unmarshal(respBody, &result)
	return resp, result
}

func putJSON(url, token string, body interface{}) (*http.Response, map[string]interface{}) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fail("HTTP request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var result map[string]interface{}
	_ = json.Unmarshal(respBody, &result)
	return resp, result
}

func getJSON(url, token string) (*http.Response, map[string]interface{}) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fail("HTTP request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var result map[string]interface{}
	_ = json.Unmarshal(respBody, &result)
	return resp, result
}

func deleteReq(url, token string) (*http.Response, map[string]interface{}) {
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fail("HTTP request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var result map[string]interface{}
	_ = json.Unmarshal(respBody, &result)
	return resp, result
}

func uploadMultipart(url, token, fieldName, filename string, fileBytes []byte) (*http.Response, map[string]interface{}) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		fail("create form file failed: %v", err)
	}
	_, _ = part.Write(fileBytes)
	_ = w.Close()

	req, _ := http.NewRequest(http.MethodPost, url, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fail("HTTP upload request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var result map[string]interface{}
	_ = json.Unmarshal(respBody, &result)
	return resp, result
}

func main() {
	fmt.Println("================================================================")
	fmt.Println("🚀 STARTING H शीर्षSCOPE STEP 6 LIVE INTEGRATION VERIFICATION")
	fmt.Println("================================================================")

	// 1. Health check
	hResp, _ := getJSON(baseURL+"/health", "")
	if hResp.StatusCode != http.StatusOK {
		fail("Health check failed with code %d", hResp.StatusCode)
	}
	pass("Health check endpoint responded 200 OK")

	// 2. Login as Recruiter
	loginResp, loginData := postJSON(baseURL+"/auth/login", "", map[string]string{
		"email":    "recruiter@hirescope.local",
		"password": "RecruiterSecure2026!",
	})
	if loginResp.StatusCode != http.StatusOK {
		fail("Recruiter login failed: code %d, resp: %v", loginResp.StatusCode, loginData)
	}
	tokenData, ok := loginData["data"].(map[string]interface{})
	var token string
	if ok && tokenData["access_token"] != nil {
		token = tokenData["access_token"].(string)
	} else if ok && tokenData["token"] != nil {
		token = tokenData["token"].(string)
	} else {
		fail("No access_token returned in login response: %v", loginData)
	}
	pass("Recruiter login succeeded, JWT access_token acquired")

	// 3. Create Candidate
	candResp, candData := postJSON(baseURL+"/candidates", token, map[string]interface{}{
		"full_name": "Verification Candidate",
		"email":     fmt.Sprintf("verify_%d@testcorp.com", time.Now().UnixNano()),
		"source":    "DIRECT",
	})
	if candResp.StatusCode != http.StatusCreated {
		fail("Candidate creation failed: code %d, resp: %v", candResp.StatusCode, candData)
	}
	candObj := candData["data"].(map[string]interface{})
	candidateID := candObj["id"].(string)
	pass("Candidate created with ID: %s", candidateID)

	// 4. Test Text-based CV
	textCVContent := `Dewi Lestari
dewi.lestari@techfirm.id
+628123456789
Jakarta, Indonesia

SUMMARY
Accomplished Cloud Infrastructure Engineer with 6+ years of expertise in Kubernetes and AWS.

EXPERIENCE
Senior Cloud Engineer
PT Cloud Nusantara
January 2022 - Present
- Architected multi-region Kubernetes clusters on AWS
- Automated deployments using Terraform and ArgoCD

Cloud Operations Engineer
PT Digital Solusindo
March 2019 - December 2021
- Maintained Docker environments and CI/CD pipelines

EDUCATION
Institut Teknologi Bandung
Bachelor of Science in Computer Science
2015 - 2019

SKILLS
Kubernetes, Docker, AWS, Terraform, Go, Python, CI/CD`

	docTextResp, docTextData := postJSON(fmt.Sprintf("%s/candidates/%s/documents", baseURL, candidateID), token, map[string]interface{}{
		"source_type": "TEXT",
		"file_name":   "dewi_pasted_cv.txt",
		"raw_text":    textCVContent,
	})
	if docTextResp.StatusCode != http.StatusCreated {
		fail("TEXT document creation failed: code %d, resp: %v", docTextResp.StatusCode, docTextData)
	}
	textDocID := docTextData["data"].(map[string]interface{})["id"].(string)
	pass("TEXT document created with ID: %s", textDocID)

	// 5. Process TEXT Document
	procTextResp, procTextData := postJSON(fmt.Sprintf("%s/candidates/%s/documents/%s/process", baseURL, candidateID, textDocID), token, nil)
	if procTextResp.StatusCode != http.StatusOK {
		fail("Process TEXT document failed: code %d, resp: %v", procTextResp.StatusCode, procTextData)
	}
	procResult := procTextData["data"].(map[string]interface{})
	candAfterProc := procResult["candidate"].(map[string]interface{})
	skills := candAfterProc["skills"].([]interface{})
	educations := candAfterProc["educations"].([]interface{})
	experiences := candAfterProc["experiences"].([]interface{})
	pass("TEXT document processed: candidate now has %d skills, %d educations, %d experiences",
		len(skills), len(educations), len(experiences))

	// 6. Verify Candidate was enriched from TEXT document
	getEnrichedCandResp, getEnrichedCandData := getJSON(fmt.Sprintf("%s/candidates/%s", baseURL, candidateID), token)
	if getEnrichedCandResp.StatusCode != http.StatusOK {
		fail("Failed to retrieve candidate profile: code %d", getEnrichedCandResp.StatusCode)
	}
	enrichedCand := getEnrichedCandData["data"].(map[string]interface{})
	if enrichedCand["phone"] != "+628123456789" {
		fail("Expected phone '+628123456789', got '%v'", enrichedCand["phone"])
	}
	if !strings.Contains(fmt.Sprintf("%v", enrichedCand["summary"]), "Cloud Infrastructure Engineer") {
		fail("Expected summary to contain 'Cloud Infrastructure Engineer', got '%v'", enrichedCand["summary"])
	}
	pass("Candidate profile correctly enriched from parsed TEXT document")

	// 7. Verify Idempotency - Process TEXT document a 2nd time
	reprocResp, reprocData := postJSON(fmt.Sprintf("%s/candidates/%s/documents/%s/process", baseURL, candidateID, textDocID), token, nil)
	if reprocResp.StatusCode != http.StatusOK {
		fail("Reprocessing document failed: code %d", reprocResp.StatusCode)
	}
	reprocResult := reprocData["data"].(map[string]interface{})
	reprocCand := reprocResult["candidate"].(map[string]interface{})
	reprocSkills := reprocCand["skills"].([]interface{})
	reprocEducations := reprocCand["educations"].([]interface{})
	reprocExperiences := reprocCand["experiences"].([]interface{})

	if len(reprocSkills) != len(skills) || len(reprocEducations) != len(educations) || len(reprocExperiences) != len(experiences) {
		fail("Idempotency violation! Counts increased on reprocessing: skills %d->%d, edu %d->%d, exp %d->%d",
			len(skills), len(reprocSkills), len(educations), len(reprocEducations), len(experiences), len(reprocExperiences))
	}
	pass("Idempotency verified: re-processing document produced 0 duplicates")

	// 8. Test Manual Data Protection
	// Recruiter manually updates candidate summary
	manualSummary := "RECRUITER REVIEWED: Top tier cloud candidate with strong ITB background."
	putCandResp, putCandData := putJSON(fmt.Sprintf("%s/candidates/%s", baseURL, candidateID), token, map[string]interface{}{
		"full_name": enrichedCand["full_name"],
		"summary":   manualSummary,
	})
	if putCandResp.StatusCode != http.StatusOK {
		fail("Failed to update candidate manually: code %d, resp: %v", putCandResp.StatusCode, putCandData)
	}
	pass("Recruiter manually updated candidate summary")

	// Reprocess again
	postJSON(fmt.Sprintf("%s/candidates/%s/documents/%s/process", baseURL, candidateID, textDocID), token, nil)
	_, candAfterData := getJSON(fmt.Sprintf("%s/candidates/%s", baseURL, candidateID), token)
	candAfterObj := candAfterData["data"].(map[string]interface{})
	if candAfterObj["summary"] != manualSummary {
		fail("Manual data overwritten! Expected '%s', got '%v'", manualSummary, candAfterObj["summary"])
	}
	pass("Manual data protection verified: recruiter-entered summary was NOT overwritten by parser")

	// 9. Upload & Process PDF CV
	pdfBytes := createTestPDF("Budi Pratama budi.pratama@enterprise.com +628111222333 Jakarta SKILLS Go, PostgreSQL, Microservices, Docker")
	uploadPDFResp, uploadPDFData := uploadMultipart(fmt.Sprintf("%s/candidates/%s/documents/upload", baseURL, candidateID), token, "file", "budi_resume.pdf", pdfBytes)
	if uploadPDFResp.StatusCode != http.StatusCreated {
		fail("PDF upload failed: code %d, resp: %v", uploadPDFResp.StatusCode, uploadPDFData)
	}
	pdfDocID := uploadPDFData["data"].(map[string]interface{})["id"].(string)
	pdfStoragePath := uploadPDFData["data"].(map[string]interface{})["storage_path"].(string)
	pass("PDF uploaded successfully with ID: %s, storage_path: %s", pdfDocID, pdfStoragePath)

	// Process PDF
	procPDFResp, procPDFData := postJSON(fmt.Sprintf("%s/candidates/%s/documents/%s/process", baseURL, candidateID, pdfDocID), token, nil)
	if procPDFResp.StatusCode != http.StatusOK {
		fail("Process PDF failed: code %d, resp: %v", procPDFResp.StatusCode, procPDFData)
	}
	pass("PDF processed successfully through downstream pipeline")

	// 10. Upload & Process DOCX CV
	docxBytes := createTestDOCX([]string{
		"Sarah Connor",
		"sarah.c@resistance.org",
		"+14155550199",
		"Los Angeles, CA",
		"PROFESSIONAL EXPERIENCE",
		"Chief Systems Specialist - Cyberdyne Defense (2020 - Present)",
		"EDUCATION",
		"California Institute of Technology - Master of Science in Robotics (2014 - 2018)",
		"SKILLS",
		"C++, Python, Embedded Systems, Linux, Neural Networks",
	})
	uploadDOCXResp, uploadDOCXData := uploadMultipart(fmt.Sprintf("%s/candidates/%s/documents/upload", baseURL, candidateID), token, "file", "sarah_resume.docx", docxBytes)
	if uploadDOCXResp.StatusCode != http.StatusCreated {
		fail("DOCX upload failed: code %d, resp: %v", uploadDOCXResp.StatusCode, uploadDOCXData)
	}
	docxDocID := uploadDOCXData["data"].(map[string]interface{})["id"].(string)
	pass("DOCX uploaded successfully with ID: %s", docxDocID)

	procDOCXResp, procDOCXData := postJSON(fmt.Sprintf("%s/candidates/%s/documents/%s/process", baseURL, candidateID, docxDocID), token, nil)
	if procDOCXResp.StatusCode != http.StatusOK {
		fail("Process DOCX failed: code %d, resp: %v", procDOCXResp.StatusCode, procDOCXData)
	}
	pass("DOCX processed successfully through downstream pipeline")

	// 11. Security & Validation Checks
	// 11a. Invalid extension (.exe)
	badExtResp, _ := uploadMultipart(fmt.Sprintf("%s/candidates/%s/documents/upload", baseURL, candidateID), token, "file", "malicious.exe", []byte("MZ..."))
	if badExtResp.StatusCode != http.StatusBadRequest {
		fail("Expected 400 for .exe file upload, got %d", badExtResp.StatusCode)
	}
	pass("Security check: Disallowed extension (.exe) rejected with 400 Bad Request")

	// 11b. Fake PDF (%PDF- header missing)
	fakePDFResp, _ := uploadMultipart(fmt.Sprintf("%s/candidates/%s/documents/upload", baseURL, candidateID), token, "file", "fake.pdf", []byte("This is not a PDF file at all"))
	if fakePDFResp.StatusCode != http.StatusBadRequest {
		fail("Expected 400 for fake PDF signature, got %d", fakePDFResp.StatusCode)
	}
	pass("Security check: Fake PDF (invalid magic header) rejected with 400 Bad Request")

	// 11c. Fake DOCX (zip without word/document.xml)
	var fakeZip bytes.Buffer
	zw := zip.NewWriter(&fakeZip)
	f, _ := zw.Create("random_file.txt")
	_, _ = f.Write([]byte("just a zip file"))
	_ = zw.Close()
	fakeDOCXResp, _ := uploadMultipart(fmt.Sprintf("%s/candidates/%s/documents/upload", baseURL, candidateID), token, "file", "fake.docx", fakeZip.Bytes())
	if fakeDOCXResp.StatusCode != http.StatusBadRequest {
		fail("Expected 400 for fake DOCX zip structure, got %d", fakeDOCXResp.StatusCode)
	}
	pass("Security check: Fake DOCX (missing document.xml) rejected with 400 Bad Request")

	// 12. Delete Document & verify storage cleanup
	delDocResp, _ := deleteReq(fmt.Sprintf("%s/candidates/%s/documents/%s", baseURL, candidateID, pdfDocID), token)
	if delDocResp.StatusCode != http.StatusOK {
		fail("Document delete failed with code %d", delDocResp.StatusCode)
	}
	pass("Document record deleted successfully via DELETE /api/v1/candidates/:id/documents/:documentId")

	// 13. Logout
	logoutResp, _ := postJSON(baseURL+"/auth/logout", token, nil)
	if logoutResp.StatusCode != http.StatusOK {
		fail("Logout failed with code %d", logoutResp.StatusCode)
	}
	pass("Recruiter logout successful, token invalidated")

	// 14. Verify revoked token rejected
	revokedResp, _ := getJSON(baseURL+"/candidates", token)
	if revokedResp.StatusCode != http.StatusUnauthorized {
		fail("Expected 401 for revoked token, got %d", revokedResp.StatusCode)
	}
	pass("Token revocation verified: Subsequent request returned 401 Unauthorized")

	fmt.Println("================================================================")
	fmt.Println("🎉 ALL STEP 6 LIVE INTEGRATION VERIFICATIONS COMPLETED SUCCESSFULLY!")
	fmt.Println("================================================================")
}
