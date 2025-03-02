package application

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/katierevinska/rpn/pkg/rpn"
)

type OrchestratorApp struct {
}

func New() *OrchestratorApp {
	return &OrchestratorApp{}
}

func (a *OrchestratorApp) RunServer() {
	http.HandleFunc("/api/v1/calculate", CalculatorHandler)
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
}
type ErrorResponse struct {
	Error string `json:"error"`
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
	res.Result = fmt.Sprintf("%f", resp)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
