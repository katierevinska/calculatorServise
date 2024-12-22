package application_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/katierevinska/rpn/internal/application"
	"github.com/katierevinska/rpn/pkg/rpn"
)

func TestRequestHandlerSuccessCase(t *testing.T) {
	expression := "1+1"
	reqBody := &application.ExpressionRequest{Expression: expression}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	application.CalculatorHandler(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Errorf("wrong status code")
	}
	expected, _ := rpn.Calc(expression)
	resp := application.SuccessResponse{Result: expected}
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

	expectedErrorResponse := application.ErrorResponse{Error: "Expression is not valid"}
	expectedData, _ := json.Marshal(expectedErrorResponse)

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			reqBody := &application.ExpressionRequest{Expression: expression}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			application.CalculatorHandler(w, req)
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
