package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hirescope/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func TestIntegration_HealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	healthHandler := handler.NewHealthHandler()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Check)
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", w.Code)
	}

	var resp handler.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("Expected status 'ok', got '%s'", resp.Status)
	}
}
