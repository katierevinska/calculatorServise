package rpn_test

import (
	"os"
	"testing"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

func TestCalc_AddsTasksWithCorrectArguments(t *testing.T) {
	os.Setenv("TIME_ADDITION_MS", "100")
	os.Setenv("TIME_SUBTRACTION_MS", "100")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "100")
	os.Setenv("TIME_DIVISIONS_MS", "100")

	tests := []struct {
		expression string
		expected   []internal.Task
	}{
		{"3+4", []internal.Task{
			{Id: "id1", Arg1: "3", Arg2: "4", Operation: "+", Operation_time: "100"},
		}},
		{"10-2*2", []internal.Task{
			{Id: "id1", Arg1: "2", Arg2: "2", Operation: "*", Operation_time: "100"},
			{Id: "id2", Arg1: "10", Arg2: "id1", Operation: "-", Operation_time: "100"},
		}},
		{"8", []internal.Task{}},
		{"2*(2.5+2)/4", []internal.Task{
			{Id: "id1", Arg1: "2.5", Arg2: "2", Operation: "+", Operation_time: "100"},
			{Id: "id2", Arg1: "2", Arg2: "id1", Operation: "*", Operation_time: "100"},
			{Id: "id3", Arg1: "id2", Arg2: "4", Operation: "/", Operation_time: "100"},
		}},
	}

	for _, tt := range tests {
		taskStore := internal.NewTaskStore()
		taskStore.Counter = *internal.NewCounter()
		_, err := rpn.Calc(tt.expression, taskStore)
		if err != nil {
			t.Errorf("Calc(%q) returned an error: %v", tt.expression, err)
		}

		if len(taskStore.GetTasks()) != len(tt.expected) {
			t.Errorf("Expected %d tasks, got %d", len(tt.expected), len(taskStore.GetTasks()))
		}

		for i, task := range taskStore.GetTasks() {
			if task.Id != tt.expected[i].Id || task.Arg1 != tt.expected[i].Arg1 || task.Arg2 != tt.expected[i].Arg2 || task.Operation != tt.expected[i].Operation || task.Operation_time != tt.expected[i].Operation_time {
				t.Errorf("Task %d: expected %+v, got %+v", i, tt.expected[i], task)
			}
		}
	}
}
