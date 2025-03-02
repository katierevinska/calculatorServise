package application

import (
	"encoding/json"
	"sync"
)

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
