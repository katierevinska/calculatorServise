package application

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

type OrchestratorApp struct {
	ExpressionStore *internal.ExpressionStore
	TaskStore       *internal.TaskStore
}

func New() *OrchestratorApp {
	return &OrchestratorApp{
		ExpressionStore: internal.NewExpressionStore(),
		TaskStore:       internal.NewTaskStore(),
	}
}

func (a *OrchestratorApp) RunServer() {
	http.HandleFunc("/api/v1/calculate", a.CalculatorHandler)
	http.HandleFunc("/api/v1/expressions", a.GetExpressionsHandler)
	http.HandleFunc("/api/v1/expressions/", a.GetExpressionByIdHandler)
	http.HandleFunc("/internal/task/new", a.GetInternalTaskHandler)
	http.HandleFunc("/internal/task", a.InternalTaskResultHandler)
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
	log.Println(r.URL.Path)
	idStr := r.URL.Path[len("/api/v1/expressions/"):]
	expression, exists := app.ExpressionStore.GetExpression(idStr)
	if !exists {
		log.Println(idStr + "Expression not found")
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expression)
}

func (app *OrchestratorApp) GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	exps := app.ExpressionStore.GetAllExpressions()
	log.Printf(r.URL.Path+"send all expressions, in map there are %d", len(exps))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(exps)
}

func (app *OrchestratorApp) InternalTaskResultHandler(w http.ResponseWriter, r *http.Request) {
	var resultData internal.TaskResult

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&resultData); err != nil {
		http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
		return
	}

	log.Printf("Получен результат задачи от агента: %s %s", resultData.Id, resultData.Result)
	app.TaskStore.TasksResStore.AddTaskRes(resultData)
	w.WriteHeader(http.StatusOK)
	exp, exist := app.ExpressionStore.GetExpression(resultData.Id)
	if exist {
		log.Printf("expression exist and resunt is set: %s %s", resultData.Id, resultData.Result)
		exp.Result = resultData.Result
		exp.Status = "calculated"
		app.ExpressionStore.AddExpression(exp)
	}
}

func (app *OrchestratorApp) GetInternalTaskHandler(w http.ResponseWriter, r *http.Request) {

	task, exists := app.TaskStore.GetFirstCorrectTask()
	if exists {
		log.Println("agent ask for task and first is " + task.Id)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(task)
		return
	}
	log.Println("agent ask for task but no one need to be solved " + task.Id)
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
	resp, err := rpn.Calc(requestExrp.Expression, app.TaskStore)
	if err != nil {
		errResp := ErrorResponse{Error: "Expression is not valid"}
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	res := new(SuccessResponse)
	res.Id = resp
	app.ExpressionStore.AddExpression(internal.Expression{ID: resp, Status: "in progress", Result: ""})
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
