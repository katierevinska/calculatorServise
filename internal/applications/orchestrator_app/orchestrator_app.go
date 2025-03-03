package application

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

type OrchestratorApp struct {
	expressionStore *internal.ExpressionStore
	taskStore       *internal.TaskStore
	taskResStore    *internal.TaskResultStore
}

func New() *OrchestratorApp {
	return &OrchestratorApp{
		expressionStore: &internal.ExpressionStore{},
		taskStore:       &internal.TaskStore{},
		taskResStore:    &internal.TaskResultStore{},
	}
}

func (a *OrchestratorApp) RunServer() {
	http.HandleFunc("/api/v1/calculate", a.CalculatorHandler)
	http.HandleFunc("/api/v1/expressions", a.GetExpressionsHandler)
	http.HandleFunc("/api/v1/expressions/", a.GetExpressionByIdHandler)
	http.HandleFunc("/internal/task/new", a.GetInternalTaskHandler)
	http.HandleFunc("/internal/task", a.InternalTasResultHandler)
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

func (app *OrchestratorApp) GetExpressionByIdHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/v1/expressions/"):]
	expression, exists := app.expressionStore.GetExpression(idStr)
	if !exists {
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expression)
}

func (app *OrchestratorApp) GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *OrchestratorApp) InternalTasResultHandler(w http.ResponseWriter, r *http.Request) {
	resultData := internal.TaskResult{}

	if err := json.NewDecoder(r.Body).Decode(&resultData); err != nil {
		http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
		return
	}
	app.taskResStore.AddTaskRes(resultData)
}

func (app *OrchestratorApp) GetInternalTaskHandler(w http.ResponseWriter, r *http.Request) {

	task, exists := app.taskStore.GetFirstTask()

	if exists {
		if value, exists := app.taskResStore.GetTaskRes(task.Arg1); exists {
			task.Arg1 = value.Result
		}
		if value, exists := app.taskResStore.GetTaskRes(task.Arg2); exists {
			task.Arg2 = value.Result
		}

		response := map[string]internal.Task{"task": task}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
	w.WriteHeader(http.StatusNotFound)
}

func (app *OrchestratorApp) CalculatorHandler(w http.ResponseWriter, r *http.Request) {
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
