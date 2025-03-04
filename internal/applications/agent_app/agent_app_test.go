package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/katierevinska/calculatorService/internal"
	application "github.com/katierevinska/calculatorService/internal/applications/agent_app"
)

func TestAgentApp_RunServer(t *testing.T) {
	tsTask := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/task/new" && r.Method == http.MethodGet {
			task := internal.Task{
				Id:             "1",
				Arg1:           "3",
				Arg2:           "4",
				Operation:      "+",
				Operation_time: "0",
			}
			if err := json.NewEncoder(w).Encode(task); err != nil {
				t.Errorf("Cannot encode task: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer tsTask.Close()

	tsResult := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/task" && r.Method == http.MethodPost {
			var result internal.TaskResult
			if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode result: %v", err)
				return
			}
			if result.Id != "1" || result.Result != "7.0000000000" {
				t.Errorf("Expected result with Id=1 and Result=7.0000000000, got %+v", result)
			}
			fmt.Fprintln(w, "OK")
		} else {
			http.NotFound(w, r)
		}
	}))
	defer tsResult.Close()

	agent := &application.AgentApp{
		OrchestratorTaskURL:   tsTask.URL + "/internal/task/new",
		OrchestratorResultURL: tsResult.URL + "/internal/task",
	}

	os.Setenv("COMPUTING_POWER", "1")

	go func() {
		agent.RunServer()
	}()

	time.Sleep(2 * time.Second)
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		task     internal.Task
		expected string
	}{
		{task: internal.Task{Arg1: "3", Arg2: "4", Operation: "+"}, expected: "7.0000000000"},
		{task: internal.Task{Arg1: "10", Arg2: "5", Operation: "-"}, expected: "5.0000000000"},
		{task: internal.Task{Arg1: "6", Arg2: "7", Operation: "*"}, expected: "42.0000000000"},
		{task: internal.Task{Arg1: "8", Arg2: "4", Operation: "/"}, expected: "2.0000000000"},
	}

	for _, tt := range tests {
		result := application.Calculate(tt.task)
		if result != tt.expected {
			t.Errorf("Expected result for %v: %s, got %s", tt.task, tt.expected, result)
		}
	}
}
