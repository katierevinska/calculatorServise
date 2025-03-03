package application

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

type OrchestratorApp struct {
}

func New() *OrchestratorApp {
	return &OrchestratorApp{}
}

func (a *OrchestratorApp) RunServer() {
	http.HandleFunc("/api/v1/calculate", CalculatorHandler)
	http.HandleFunc("/api/v1/expressions", GetExpressionsHandler)
	http.HandleFunc("/api/v1/expressions/:id", GetExpressionByIdHandler)
	http.HandleFunc("/internal/task", GetInternalTaskHandler)
	http.HandleFunc("/internal/task", InternalTasResultHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

type ExpressionRequest struct {
	Expression string `json:"expression"`
}
type SuccessResponse struct {
	Id string `json:"id"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

func GetExpressionByIdHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/v1/expressions/"):]
	store := &internal.ExpressionStore{}
	expression, exists := store.GetExpression(idStr)
	if !exists {
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expression)
}

func GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {

}

func InternalTasResultHandler(w http.ResponseWriter, r *http.Request) {
	resultData := internal.TaskResult{}

	if err := json.NewDecoder(r.Body).Decode(&resultData); err != nil {
		http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
		return
	}
	store := &internal.TaskResultStore{}
	store.AddTaskRes(resultData)
}

func GetInternalTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskStore := &internal.TaskStore{}
	taskResStore := &internal.TaskResultStore{}

	task := taskStore.GetFirstTask()

	if value, exists := taskResStore.GetTaskRes(task.Arg1); exists {
		task.Arg1 = value.Result
	}
	if value, exists := taskResStore.GetTaskRes(task.Arg2); exists {
		task.Arg2 = value.Result
	}

	response := map[string]internal.Task{"task": task}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func CalculatorHandler(w http.ResponseWriter, r *http.Request) {
	requestExrp := new(ExpressionRequest)
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&requestExrp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := rpn.Calc(requestExrp.Expression)
	if err != nil {
		errResp := ErrorResponse{Error: "Expression is not valid"}
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	res := new(SuccessResponse)
	res.Id = resp
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
