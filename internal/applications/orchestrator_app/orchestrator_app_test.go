package application_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/katierevinska/calculatorService/internal"
	application "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
)

func TestOrchestratorApp_CalculatorHandler(t *testing.T) {
	app := application.New()

	os.Setenv("TIME_ADDITION_MS", "100")
	os.Setenv("TIME_SUBTRACTION_MS", "100")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "100")
	os.Setenv("TIME_DIVISIONS_MS", "100")

	tests := []struct {
		expression     string
		expectedStatus int
		expectedResult string
	}{
		{"3+4", http.StatusOK, "id1"},
		{"10-2", http.StatusOK, "id2"},
		{"5*3", http.StatusOK, "id3"},
		{"8/4", http.StatusOK, "id4"},
		{"34+", http.StatusUnprocessableEntity, ""},
		{"3&4", http.StatusUnprocessableEntity, ""},
	}

	for _, tt := range tests {
		reqBody, _ := json.Marshal(application.ExpressionRequest{Expression: tt.expression})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		app.CalculatorHandler(w, req)
		if w.Code != tt.expectedStatus {
			t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
		}
		if tt.expectedStatus == http.StatusOK {
			var successResp application.SuccessResponse
			if err := json.NewDecoder(w.Body).Decode(&successResp); err != nil {
				t.Errorf("Failed to decode response: %v", err)
			}
			if successResp.Id != tt.expectedResult {
				t.Errorf("Expected result id %s, got %s", tt.expectedResult, successResp.Id)
			}
		} else {
			var errorResp application.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
				t.Errorf("Failed to decode error response: %v", err)
			}
			if errorResp.Error != "Expression is not valid" {
				t.Errorf("Expected error message 'Expression is not valid', got %s", errorResp.Error)
			}
		}
	}

	if len(app.ExpressionStore.GetAllExpressions()) == 0 {
		t.Error("Expected expressions to be stored, but found none.")
	}
}

func TestOrchestratorApp_InternalTaskResultHandler(t *testing.T) {
	app := application.New()
	app.ExpressionStore.AddExpression(internal.Expression{ID: "id1"})
	result := internal.TaskResult{Id: "id1", Result: "7.0000000000"}
	body, _ := json.Marshal(result)

	req := httptest.NewRequest(http.MethodPost, "/internal/task", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.InternalTaskResultHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if exp, exists := app.ExpressionStore.GetExpression("id1"); !exists || exp.Result != "7.0000000000" {
		t.Errorf("Expected expression with Id 'id1' to have result '7.0000000000', got %v", exp)
	}
}

func TestOrchestratorApp_GetExpressionsHandler(t *testing.T) {
	app := application.New()
	app.ExpressionStore.AddExpression(internal.Expression{ID: "id1", Status: "calculated", Result: "7.0000000000"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expressions", nil)
	w := httptest.NewRecorder()

	app.GetExpressionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var expressions []internal.Expression
	if err := json.NewDecoder(w.Body).Decode(&expressions); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	if len(expressions) == 0 {
		t.Error("Expected expressions to be returned, but found none.")
	}
}
