package application_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orchestratorApp "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

func TestRequestHandlerSuccessCase(t *testing.T) {
	expression := "1+1"
	reqBody := &orchestratorApp.ExpressionRequest{Expression: expression}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/calculate", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	orchestratorApp.CalculatorHandler(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Errorf("wrong status code")
	}
	expected, _ := rpn.Calc(expression)
	resp := orchestratorApp.SuccessResponse{Result: fmt.Sprintf("%f", expected)}
	expectedData, _ := json.Marshal(resp)
	if strings.TrimSpace(string(data)) != strings.TrimSpace(string(expectedData)) {
		t.Errorf("wrong result by server: got %v, want %v", string(data), string(expectedData))
	}
}

func TestRequestHandlerBadRequestCase(t *testing.T) {
	expressions := []string{
		"1+1+",
		"2/0",
		"3j",
	}

	expectedErrorResponse := orchestratorApp.ErrorResponse{Error: "Expression is not valid"}
	expectedData, _ := json.Marshal(expectedErrorResponse)

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			reqBody := &orchestratorApp.ExpressionRequest{Expression: expression}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v1/calculate", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			orchestratorApp.CalculatorHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			data, _ := io.ReadAll(res.Body)

			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("wrong status code: got %v want %v", res.StatusCode, http.StatusUnprocessableEntity)
			}

			if strings.TrimSpace(string(data)) != strings.TrimSpace(string(expectedData)) {
				t.Errorf("wrong result by server: got %v, want %v", string(data), string(expectedData))
			}
		})
	}
}
