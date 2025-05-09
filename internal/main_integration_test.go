package internal_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"strings"
	"testing"
	"time"

	"github.com/katierevinska/calculatorService/internal"
	orchestratorApp "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
	"github.com/katierevinska/calculatorService/internal/auth"
	"github.com/katierevinska/calculatorService/internal/database"
	"github.com/katierevinska/calculatorService/internal/middleware"
	"github.com/katierevinska/calculatorService/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration_ExpressionCalculation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full integration test in short mode.")
	}

	os.Setenv("TIME_ADDITION_MS", "50")
	os.Setenv("TIME_SUBTRACTION_MS", "50")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "50")
	os.Setenv("TIME_DIVISIONS_MS", "50")
	os.Setenv("JWT_SECRET", "integrationtestsecret")

	err := auth.InitJWT()
	require.NoError(t, err)

	db, err := database.InitDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	orchApp := orchestratorApp.New(db)

	testOrchestrator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/register" && r.Method == http.MethodPost:
			orchApp.RegisterUserHandler(w, r)
		case r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost:
			orchApp.LoginUserHandler(w, r)
		case r.URL.Path == "/api/v1/calculate" && r.Method == http.MethodPost:
			middleware.AuthMiddleware(http.HandlerFunc(orchApp.CalculatorHandler)).ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/expressions"):
			if r.Method == http.MethodGet {
				pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
				if len(pathParts) == 4 && pathParts[3] != "" {
					middleware.AuthMiddleware(http.HandlerFunc(orchApp.GetExpressionByIdHandler)).ServeHTTP(w, r)
				} else if len(pathParts) == 3 {
					middleware.AuthMiddleware(http.HandlerFunc(orchApp.GetExpressionsHandler)).ServeHTTP(w, r)
				} else {
					http.NotFound(w, r)
				}
			} else {
				http.NotFound(w, r)
			}
		case r.URL.Path == "/internal/task/new" && r.Method == http.MethodGet:
			orchApp.GetInternalTaskHandler(w, r)
		case r.URL.Path == "/internal/task" && r.Method == http.MethodPost:
			orchApp.InternalTaskResultHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer testOrchestrator.Close()
	log.Printf("Test Orchestrator running at: %s", testOrchestrator.URL)

	os.Setenv("COMPUTING_POWER", "1")

	go func() {
		log.Println("Starting test agent worker goroutine")
		log.Println("Agent part of integration test is conceptual due to infinite loop in agent.RunServer()")
	}()

	regClient := testOrchestrator.Client()
	regReqBody, _ := json.Marshal(models.Credentials{Login: "integuser", Password: "integpassword"})
	regResp, err := regClient.Post(testOrchestrator.URL+"/api/v1/register", "application/json", bytes.NewBuffer(regReqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, regResp.StatusCode, "Registration failed")
	regResp.Body.Close()

	loginReqBody, _ := json.Marshal(models.Credentials{Login: "integuser", Password: "integpassword"})
	loginResp, err := regClient.Post(testOrchestrator.URL+"/api/v1/login", "application/json", bytes.NewBuffer(loginReqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "Login failed")
	var tokenResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(loginResp.Body).Decode(&tokenResp)
	require.NoError(t, err)
	loginResp.Body.Close()
	userToken := tokenResp.Token
	require.NotEmpty(t, userToken)

	exprToCalc := "2*3+4"
	calcReqBody, _ := json.Marshal(orchestratorApp.ExpressionRequest{Expression: exprToCalc})
	calcReq, _ := http.NewRequest(http.MethodPost, testOrchestrator.URL+"/api/v1/calculate", bytes.NewBuffer(calcReqBody))
	calcReq.Header.Set("Authorization", "Bearer "+userToken)
	calcReq.Header.Set("Content-Type", "application/json")
	calcResp, err := regClient.Do(calcReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, calcResp.StatusCode)
	var successResp orchestratorApp.SuccessResponse
	err = json.NewDecoder(calcResp.Body).Decode(&successResp)
	require.NoError(t, err)
	calcResp.Body.Close()
	expressionID := successResp.Id
	require.NotEmpty(t, expressionID)

	getTaskResp1, err := regClient.Get(testOrchestrator.URL + "/internal/task/new")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getTaskResp1.StatusCode)
	var task1 internal.Task
	json.NewDecoder(getTaskResp1.Body).Decode(&task1)
	getTaskResp1.Body.Close()
	assert.Equal(t, "2", task1.Arg1)
	assert.Equal(t, "3", task1.Arg2)
	assert.Equal(t, "*", task1.Operation)

	task1Result := internal.TaskResult{Id: task1.Id, Result: "6.0000000000"}
	task1ResultBody, _ := json.Marshal(task1Result)
	postResult1Resp, err := regClient.Post(testOrchestrator.URL+"/internal/task", "application/json", bytes.NewBuffer(task1ResultBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, postResult1Resp.StatusCode)
	postResult1Resp.Body.Close()

	getTaskResp2, err := regClient.Get(testOrchestrator.URL + "/internal/task/new")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getTaskResp2.StatusCode)
	var task2 internal.Task
	json.NewDecoder(getTaskResp2.Body).Decode(&task2)
	getTaskResp2.Body.Close()
	assert.Equal(t, expressionID, task2.Id)
	assert.Equal(t, task1Result.Result, task2.Arg1)
	assert.Equal(t, "4", task2.Arg2)
	assert.Equal(t, "+", task2.Operation)

	task2Result := internal.TaskResult{Id: task2.Id, Result: "10.0000000000"}
	task2ResultBody, _ := json.Marshal(task2Result)
	postResult2Resp, err := regClient.Post(testOrchestrator.URL+"/internal/task", "application/json", bytes.NewBuffer(task2ResultBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, postResult2Resp.StatusCode)
	postResult2Resp.Body.Close()

	time.Sleep(200 * time.Millisecond)

	getExprReq, _ := http.NewRequest(http.MethodGet, testOrchestrator.URL+"/api/v1/expressions/"+expressionID, nil)
	getExprReq.Header.Set("Authorization", "Bearer "+userToken)
	getExprResp, err := regClient.Do(getExprReq)
	require.NoError(t, err)

	bodyBytes, _ := io.ReadAll(getExprResp.Body)
	getExprResp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	require.Equal(t, http.StatusOK, getExprResp.StatusCode, "Failed to get expression. Body: "+string(bodyBytes))

	var finalExpression internal.Expression
	err = json.NewDecoder(getExprResp.Body).Decode(&finalExpression)
	require.NoError(t, err, "Failed to decode final expression. Body: "+string(bodyBytes))
	getExprResp.Body.Close()

	assert.Equal(t, "calculated", finalExpression.Status)
	assert.Equal(t, "10.0000000000", finalExpression.Result)
}
