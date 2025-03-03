package internal

import (
	"encoding/json"
	"sync"
)

type Task struct {
	Id             string `json:"id"`             //идентификатор задачи,
	Arg1           string `json:"arg1"`           //имя первого аргумента,
	Arg2           string `json:"arg2"`           //имя второго аргумента,
	Operation      string `json:"operation"`      //операция,
	Operation_time string `json:"operation_time"` //время выполнения операции
}

type TaskResult struct {
	Id     string `json:"id"`
	Result string `json:"result"`
}

type Expression struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type ExpressionStore struct {
	expressions map[string]Expression
	mu          sync.Mutex
}

func (store *ExpressionStore) AddExpression(expr Expression) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expressions[expr.ID] = expr
}

func (store *ExpressionStore) GetExpression(id string) (Expression, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	expr, exists := store.expressions[id]
	return expr, exists
}

func (store *ExpressionStore) ToJSON() (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	jsonData, err := json.Marshal(store.expressions)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

type TaskStore struct {
	tasks   []Task
	counter int64
	mu      sync.Mutex
}

func (ts *TaskStore) GetCounter() int64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.counter += 1
	return ts.counter
}

func (store *TaskStore) AddTask(t Task) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.tasks = append(store.tasks, t)
}

func (store *TaskStore) GetTasks() []Task {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.tasks
}
func (store *TaskStore) GetFirstTask() (Task, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.tasks) == 0 {
		return Task{}, false
	}
	task := store.tasks[0]
	store.tasks = store.tasks[1:]
	return task, true
}
func (store *TaskStore) ToJSON() (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	jsonData, err := json.Marshal(store.tasks)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

type TaskResultStore struct {
	tasksRes map[string]TaskResult
	mu       sync.Mutex
}

func (store *TaskResultStore) AddTaskRes(t TaskResult) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.tasksRes[t.Id] = t
}

func (store *TaskResultStore) GetTaskRes(id string) (TaskResult, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	task, exists := store.tasksRes[id]
	return task, exists
}
