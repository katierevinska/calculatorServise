package application

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
	Result string `json:"result"`
	//Id string `json:"id"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

func GetExpressionByIdHandler(w http.ResponseWriter, r *http.Request) {

}

func GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {

}

func InternalTasResultHandler(w http.ResponseWriter, r *http.Request) {

}

func GetInternalTaskHandler(w http.ResponseWriter, r *http.Request) {

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
	//add simple expressions to the map
	res := new(SuccessResponse)
	//res.id = ...
	res.Result = fmt.Sprintf("%f", resp)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
