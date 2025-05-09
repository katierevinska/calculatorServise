package rpn_test

import (
	"os"
	"testing"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/internal/store"
	"github.com/katierevinska/calculatorService/pkg/rpn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEnvForRPN() {
	os.Setenv("TIME_ADDITION_MS", "10")
	os.Setenv("TIME_SUBTRACTION_MS", "10")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "20")
	os.Setenv("TIME_DIVISIONS_MS", "20")
}

func TestCalc_AddsTasksWithCorrectArguments(t *testing.T) {
	setupEnvForRPN()

	tests := []struct {
		name           string
		expression     string
		expectedTasks  []internal.Task
		expectedLastID string
		expectError    bool
	}{
		{
			name:       "Simple addition",
			expression: "3+4",
			expectedTasks: []internal.Task{
				{Id: "id1", Arg1: "3", Arg2: "4", Operation: "+", Operation_time: "10"},
			},
			expectedLastID: "id1",
		},
		{
			name:       "Subtraction and multiplication with precedence",
			expression: "10-2*2",
			expectedTasks: []internal.Task{
				{Id: "id1", Arg1: "2", Arg2: "2", Operation: "*", Operation_time: "20"},
				{Id: "id2", Arg1: "10", Arg2: "id1", Operation: "-", Operation_time: "10"},
			},
			expectedLastID: "id2",
		},
		{
			name:           "Single number",
			expression:     "8",
			expectedTasks:  []internal.Task{},
			expectedLastID: "8",
		},
		{
			name:       "Complex with parentheses",
			expression: "2*(2.5+2)/4",
			expectedTasks: []internal.Task{
				{Id: "id1", Arg1: "2.5", Arg2: "2", Operation: "+", Operation_time: "10"},
				{Id: "id2", Arg1: "2", Arg2: "id1", Operation: "*", Operation_time: "20"},
				{Id: "id3", Arg1: "id2", Arg2: "4", Operation: "/", Operation_time: "20"},
			},
			expectedLastID: "id3",
		},
		{
			name:        "Division by zero",
			expression:  "1/0",
			expectError: true,
		},
		{
			name:        "Invalid expression - unmatched parenthesis",
			expression:  "(2+3",
			expectError: true,
		},
		{
			name:        "Invalid expression - operator start",
			expression:  "*2+3",
			expectError: true,
		},
		{
			name:        "Invalid expression - unknown symbol",
			expression:  "2%3",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStore := store.NewTaskStore()

			lastID, err := rpn.Calc(tt.expression, taskStore)

			if tt.expectError {
				require.Error(t, err, "Calc(%q) should have returned an error", tt.expression)
				return
			}
			require.NoError(t, err, "Calc(%q) returned an unexpected error: %v", tt.expression, err)

			assert.Equal(t, tt.expectedLastID, lastID, "Unexpected last task ID")

			actualTasks := taskStore.GetTasks()
			require.Len(t, actualTasks, len(tt.expectedTasks), "Expected %d tasks, got %d", len(tt.expectedTasks), len(actualTasks))

			for i, expectedTask := range tt.expectedTasks {
				actualTask := actualTasks[i]
				assert.Equal(t, expectedTask.Id, actualTask.Id, "Task %d: ID mismatch", i)
				assert.Equal(t, expectedTask.Arg1, actualTask.Arg1, "Task %d: Arg1 mismatch", i)
				assert.Equal(t, expectedTask.Arg2, actualTask.Arg2, "Task %d: Arg2 mismatch", i)
				assert.Equal(t, expectedTask.Operation, actualTask.Operation, "Task %d: Operation mismatch", i)
				assert.Equal(t, expectedTask.Operation_time, actualTask.Operation_time, "Task %d: Operation_time mismatch", i)
			}
		})
	}
}
