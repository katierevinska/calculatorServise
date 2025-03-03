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
	expressions []Expression
	mu          sync.Mutex
}

func (store *ExpressionStore) AddExpression(expr Expression) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expressions = append(store.expressions, expr)
}

func (store *ExpressionStore) GetExpressions() []Expression {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.expressions
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
