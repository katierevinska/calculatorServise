package application_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/katierevinska/calculatorService/internal"
	agentapp "github.com/katierevinska/calculatorService/internal/applications/agent_app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentApp_RunServer_FetchesAndSendsResult(t *testing.T) {
	taskSent := false
	resultReceived := false
	var wg sync.WaitGroup
	wg.Add(1)

	mockOrchestrator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/task/new" && r.Method == http.MethodGet {
			if !taskSent {
				task := internal.Task{
					Id:             "task1",
					Arg1:           "10",
					Arg2:           "5",
					Operation:      "+",
					Operation_time: "10",
				}
				json.NewEncoder(w).Encode(task)
				taskSent = true
			} else {
				http.NotFound(w, r)
			}
		} else if r.URL.Path == "/internal/task" && r.Method == http.MethodPost {
			var res internal.TaskResult
			err := json.NewDecoder(r.Body).Decode(&res)
			require.NoError(t, err)
			assert.Equal(t, "task1", res.Id)
			assert.Equal(t, "15.0000000000", res.Result)
			resultReceived = true
			w.WriteHeader(http.StatusOK)
			wg.Done()
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockOrchestrator.Close()

	agent := agentapp.New()
	agent.OrchestratorTaskURL = mockOrchestrator.URL + "/internal/task/new"
	agent.OrchestratorResultURL = mockOrchestrator.URL + "/internal/task"

	os.Setenv("COMPUTING_POWER", "1")

	go agent.RunServer()

	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		assert.True(t, taskSent, "Agent should have fetched a task")
		assert.True(t, resultReceived, "Agent should have sent a result")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out. Agent did not send result or orchestrator mock failed.")
	}
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		name     string
		task     internal.Task
		expected string
	}{
		{name: "Addition", task: internal.Task{Arg1: "3", Arg2: "4", Operation: "+"}, expected: "7.0000000000"},
		{name: "Subtraction", task: internal.Task{Arg1: "10", Arg2: "5", Operation: "-"}, expected: "5.0000000000"},
		{name: "Multiplication", task: internal.Task{Arg1: "6", Arg2: "7", Operation: "*"}, expected: "42.0000000000"},
		{name: "Division", task: internal.Task{Arg1: "8", Arg2: "4", Operation: "/"}, expected: "2.0000000000"},
		{name: "Division with float result", task: internal.Task{Arg1: "1", Arg2: "3", Operation: "/"}, expected: "0.3333333333"},
		{name: "Invalid number arg1", task: internal.Task{Arg1: "abc", Arg2: "4", Operation: "+"}, expected: "Error: Invalid number"},
		{name: "Invalid number arg2", task: internal.Task{Arg1: "3", Arg2: "xyz", Operation: "+"}, expected: "Error: Invalid number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentapp.Calculate(tt.task) // Use package alias
			assert.Equal(t, tt.expected, result)
		})
	}
}
