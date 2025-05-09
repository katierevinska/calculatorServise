package internal

type Task struct {
	Id             string `json:"id"`
	Arg1           string `json:"arg1"`
	Arg2           string `json:"arg2"`
	Operation      string `json:"operation"`
	Operation_time string `json:"operation_time"`
}

type TaskResult struct {
	Id     string `json:"id"`
	Result string `json:"result"`
}

type Expression struct {
	ID               string `json:"id"`
	UserID           int64  `json:"-"`
	ExpressionString string `json:"expression"`
	Status           string `json:"status"`
	Result           string `json:"result,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
}
