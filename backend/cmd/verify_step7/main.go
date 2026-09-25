package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func main() {
	fmt.Println("================================================================")
	fmt.Println("🚀 STARTING HIRESCOPE STEP 7 LIVE INTEGRATION VERIFICATION")
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
	} else {
		fail("No access_token returned in login response")
	}
	pass("Recruiter login succeeded, JWT access_token acquired")

	// 3. Create Job Vacancy
	jobResp, jobData := postJSON(baseURL+"/jobs", token, map[string]interface{}{
		"title":           "Principal Systems Engineer",
		"description":     "Core infrastructure role requiring Go and PostgreSQL expertise.",
		"department":      "Engineering",
		"location":        "Jakarta, Indonesia",
		"employment_type": "FULL_TIME",
	})
	if jobResp.StatusCode != http.StatusCreated {
		fail("Job creation failed: code %d, resp: %v", jobResp.StatusCode, jobData)
	}
	jobObj := jobData["data"].(map[string]interface{})
	jobID := jobObj["id"].(string)
	pass("Job created with ID: %s", jobID)

	// Add Requirements to Job:
	// 4 REQUIRED, 2 PREFERRED
	reqDefs := []map[string]interface{}{
		{"category": "SKILL", "requirement": "Go", "importance": "REQUIRED"},
		{"category": "SKILL", "requirement": "PostgreSQL", "importance": "REQUIRED"},
		{"category": "EXPERIENCE", "requirement": "3 years experience", "importance": "REQUIRED"},
		{"category": "EDUCATION", "requirement": "Bachelor's degree in Computer Science", "importance": "REQUIRED"},
		{"category": "SKILL", "requirement": "Docker", "importance": "PREFERRED"},
		{"category": "LANGUAGE", "requirement": "English", "importance": "PREFERRED"},
	}

	for _, rDef := range reqDefs {
		rResp, rData := postJSON(fmt.Sprintf("%s/jobs/%s/requirements", baseURL, jobID), token, rDef)
		if rResp.StatusCode != http.StatusCreated {
			fail("Failed creating requirement %v: code %d, resp: %v", rDef["requirement"], rResp.StatusCode, rData)
		}
	}
	pass("Configured 4 REQUIRED and 2 PREFERRED requirements on job")

	// 4. Create Candidate
	candResp, candData := postJSON(baseURL+"/candidates", token, map[string]interface{}{
		"full_name": "Hendra Wijaya",
		"email":     fmt.Sprintf("hendra_%d@cloudnative.id", time.Now().UnixNano()),
		"headline":  "Senior Backend Developer",
		"summary":   "Passionate Go and cloud developer. English — Professional Working Proficiency.",
		"source":    "DIRECT",
	})
	if candResp.StatusCode != http.StatusCreated {
		fail("Candidate creation failed: code %d, resp: %v", candResp.StatusCode, candData)
	}
	candObj := candData["data"].(map[string]interface{})
	candidateID := candObj["id"].(string)
	pass("Candidate created with ID: %s", candidateID)

	// Add candidate skills
	postJSON(fmt.Sprintf("%s/candidates/%s/skills", baseURL, candidateID), token, map[string]string{"skill": "Go"})
	postJSON(fmt.Sprintf("%s/candidates/%s/skills", baseURL, candidateID), token, map[string]string{"skill": "PostgreSQL"})
	postJSON(fmt.Sprintf("%s/candidates/%s/skills", baseURL, candidateID), token, map[string]string{"skill": "Git"})

	// Add candidate education
	postJSON(fmt.Sprintf("%s/candidates/%s/educations", baseURL, candidateID), token, map[string]interface{}{
		"institution":    "Institut Teknologi Bandung",
		"degree":         "Bachelor of Computer Science",
		"field_of_study": "Computer Science",
	})

	// Add candidate experience (3 years 6 months)
	startDate := time.Now().UTC().AddDate(-3, -6, 0).Format("2006-01-02")
	postJSON(fmt.Sprintf("%s/candidates/%s/experiences", baseURL, candidateID), token, map[string]interface{}{
		"company":         "PT Cloud Nusantara",
		"position":        "Backend Developer",
		"employment_type": "FULL_TIME",
		"start_date":      startDate,
		"is_current":      true,
		"description":     "Microservice engineering with Go and high-scale PostgreSQL",
	})
	pass("Enriched candidate with skills, education, and 3.5 years of experience")

	// 5. Test Relationship Guard: Screening unassociated candidate MUST fail
	unassocResp, unassocData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screen", baseURL, jobID, candidateID), token, nil)
	if unassocResp.StatusCode != http.StatusBadRequest {
		fail("Expected 400 Bad Request for unassociated candidate, got %d: %v", unassocResp.StatusCode, unassocData)
	}
	pass("Relationship security verified: unassociated candidate rejected with 400 Bad Request")

	// 6. Link candidate to job
	linkResp, linkData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates", baseURL, jobID), token, map[string]string{
		"candidate_id": candidateID,
	})
	if linkResp.StatusCode != http.StatusCreated {
		fail("Linking candidate to job failed: code %d, resp: %v", linkResp.StatusCode, linkData)
	}
	pass("Candidate successfully linked to job vacancy")

	// 7. Execute Screening
	screenResp, screenData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screen", baseURL, jobID, candidateID), token, nil)
	if screenResp.StatusCode != http.StatusOK {
		fail("Candidate screening failed: code %d, resp: %v", screenResp.StatusCode, screenData)
	}
	screenPayload := screenData["data"].(map[string]interface{})
	screeningRes := screenPayload["screening_result"].(map[string]interface{})
	matches := screenPayload["matches"].([]interface{})

	// Verify Overall Status is QUALIFIED
	if screeningRes["status"] != "QUALIFIED" {
		fail("Expected overall status QUALIFIED, got '%v'", screeningRes["status"])
	}
	if screeningRes["required_match_count"].(float64) != 4 {
		fail("Expected 4 required matches, got %v", screeningRes["required_match_count"])
	}
	if screeningRes["required_mismatch_count"].(float64) != 0 {
		fail("Expected 0 required mismatches, got %v", screeningRes["required_mismatch_count"])
	}
	if screeningRes["preferred_match_count"].(float64) != 1 {
		fail("Expected 1 preferred match (English), got %v", screeningRes["preferred_match_count"])
	}
	if screeningRes["preferred_unknown_count"].(float64) != 1 {
		fail("Expected 1 preferred unknown (Docker absence), got %v", screeningRes["preferred_unknown_count"])
	}
	pass("Screening completed successfully! Status = QUALIFIED (All 4 REQUIRED satisfied, 0 Mismatch)")

	// Verify explainability of individual requirement matches
	for _, mRaw := range matches {
		m := mRaw.(map[string]interface{})
		reqName := m["requirement"].(string)
		status := m["status"].(string)
		evidence := m["evidence"].(string)
		reason := m["reason"].(string)
		confidence := m["confidence"].(string)
		src := m["source"].(string)

		if strings.TrimSpace(reason) == "" {
			fail("Explainability violation: Requirement '%s' has empty reason", reqName)
		}
		if confidence == "" || src == "" {
			fail("Missing metadata on requirement '%s': confidence=%s, source=%s", reqName, confidence, src)
		}

		if reqName == "Docker" && status != "UNKNOWN" {
			fail("Absence of Docker should be UNKNOWN, got %s", status)
		}
		if reqName == "English" && status != "MATCH" {
			fail("English proficiency should be MATCH, got %s", status)
		}
		fmt.Printf("   ├─ [%s] %s -> %s (Source: %s, Conf: %s)\n      Evidence: %s\n",
			m["importance"], reqName, status, src, confidence, evidence)
	}
	pass("All requirement matches verified for verifiable evidence and clear reasoning")

	// 8. Test GET latest screening endpoint
	getScreenResp, getScreenData := getJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screening", baseURL, jobID, candidateID), token)
	if getScreenResp.StatusCode != http.StatusOK {
		fail("GET latest screening failed: code %d, resp: %v", getScreenResp.StatusCode, getScreenData)
	}
	getPayload := getScreenData["data"].(map[string]interface{})
	getResultObj := getPayload["screening_result"].(map[string]interface{})
	if getResultObj["id"] != screeningRes["id"] {
		fail("GET screening returned different result ID: %v vs %v", getResultObj["id"], screeningRes["id"])
	}
	pass("GET /api/v1/jobs/:id/candidates/:candidateId/screening retrieved identical latest result")

	// 9. Test Idempotency: Re-screening must update existing result and not duplicate matches
	rescreenResp, rescreenData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screen", baseURL, jobID, candidateID), token, nil)
	if rescreenResp.StatusCode != http.StatusOK {
		fail("Re-screening failed: code %d, resp: %v", rescreenResp.StatusCode, rescreenData)
	}
	rescreenPayload := rescreenData["data"].(map[string]interface{})
	rescreenResult := rescreenPayload["screening_result"].(map[string]interface{})
	rescreenMatches := rescreenPayload["matches"].([]interface{})

	if rescreenResult["id"] != screeningRes["id"] {
		fail("Idempotency violation: new result ID created on re-screening (%v vs %v)", rescreenResult["id"], screeningRes["id"])
	}
	if len(rescreenMatches) != len(matches) {
		fail("Idempotency violation: matches count changed on re-screening (%d vs %d)", len(rescreenMatches), len(matches))
	}
	pass("Idempotency verified: Re-screening updated existing record with 0 duplicate matches")

	// 10. Test Overall Status: REQUIRED Mismatch -> NOT_QUALIFIED
	// Add a REQUIRED requirement for Master's degree (candidate only has Bachelor)
	addMasterResp, addMasterData := postJSON(fmt.Sprintf("%s/jobs/%s/requirements", baseURL, jobID), token, map[string]interface{}{
		"category":    "EDUCATION",
		"requirement": "Master's degree",
		"importance":  "REQUIRED",
	})
	if addMasterResp.StatusCode != http.StatusCreated {
		fail("Failed to add Master requirement: %v", addMasterData)
	}
	masterReqID := addMasterData["data"].(map[string]interface{})["id"].(string)

	screenMismatchResp, screenMismatchData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screen", baseURL, jobID, candidateID), token, nil)
	if screenMismatchResp.StatusCode != http.StatusOK {
		fail("Screening with mismatch failed: %v", screenMismatchData)
	}
	mismatchResult := screenMismatchData["data"].(map[string]interface{})["screening_result"].(map[string]interface{})
	if mismatchResult["status"] != "NOT_QUALIFIED" {
		fail("Expected status NOT_QUALIFIED when REQUIRED requirement is mismatched, got '%v'", mismatchResult["status"])
	}
	if mismatchResult["required_mismatch_count"].(float64) != 1 {
		fail("Expected 1 required mismatch, got %v", mismatchResult["required_mismatch_count"])
	}
	pass("Overall status logic verified: REQUIRED education mismatch correctly transitions status to NOT_QUALIFIED")

	// 11. Test PREFERRED mismatch does NOT disqualify candidate
	// Update Master requirement importance to PREFERRED
	putJSON(fmt.Sprintf("%s/jobs/%s/requirements/%s", baseURL, jobID, masterReqID), token, map[string]interface{}{
		"category":    "EDUCATION",
		"requirement": "Master's degree",
		"importance":  "PREFERRED",
	})

	screenPrefResp, screenPrefData := postJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screen", baseURL, jobID, candidateID), token, nil)
	if screenPrefResp.StatusCode != http.StatusOK {
		fail("Screening with preferred mismatch failed: %v", screenPrefData)
	}
	prefResult := screenPrefData["data"].(map[string]interface{})["screening_result"].(map[string]interface{})
	if prefResult["status"] != "QUALIFIED" {
		fail("Preferred mismatch must NOT disqualify candidate! Expected QUALIFIED, got '%v'", prefResult["status"])
	}
	if prefResult["preferred_mismatch_count"].(float64) != 1 {
		fail("Expected 1 preferred mismatch count, got %v", prefResult["preferred_mismatch_count"])
	}
	pass("Preferred requirement rule verified: PREFERRED mismatch recorded without disqualifying candidate")

	// 12. Test Logout & Token Revocation
	logoutResp, _ := postJSON(baseURL+"/auth/logout", token, nil)
	if logoutResp.StatusCode != http.StatusOK {
		fail("Logout failed with code %d", logoutResp.StatusCode)
	}
	pass("Recruiter logged out, token revoked")

	revokedResp, _ := getJSON(fmt.Sprintf("%s/jobs/%s/candidates/%s/screening", baseURL, jobID, candidateID), token)
	if revokedResp.StatusCode != http.StatusUnauthorized {
		fail("Expected 401 Unauthorized for revoked token, got %d", revokedResp.StatusCode)
	}
	pass("Authentication security verified: Revoked token rejected with 401 Unauthorized")

	fmt.Println("================================================================")
	fmt.Println("🎉 ALL STEP 7 LIVE INTEGRATION VERIFICATIONS COMPLETED SUCCESSFULLY!")
	fmt.Println("================================================================")
}
