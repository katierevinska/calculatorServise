package orchestrator_app_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/katierevinska/calculatorService/internal"
	orchestratorApp "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
	"github.com/katierevinska/calculatorService/internal/auth"
	"github.com/katierevinska/calculatorService/internal/database"
	"github.com/katierevinska/calculatorService/internal/middleware"
	"github.com/katierevinska/calculatorService/internal/models"
	store "github.com/katierevinska/calculatorService/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

var testDB *sql.DB
var testApp *orchestratorApp.OrchestratorApp
var testUserToken string
var testUserID int64

func setupTestApp(t *testing.T) func() {
	t.Helper()

	os.Setenv("TIME_ADDITION_MS", "10")
	os.Setenv("TIME_SUBTRACTION_MS", "10")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "20")
	os.Setenv("TIME_DIVISIONS_MS", "20")
	os.Setenv("JWT_SECRET", "testsecretforserver")

	err := auth.InitJWT()
	require.NoError(t, err, "Failed to init JWT")

	db, err := database.InitDB(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	testDB = db

	testApp = orchestratorApp.New(testDB)

	testUsername := "testuser"
	testPassword := "testpassword"

	regReqBody, _ := json.Marshal(models.Credentials{Login: testUsername, Password: testPassword})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBuffer(regReqBody))
	regRec := httptest.NewRecorder()
	testApp.RegisterUserHandler(regRec, regReq)
	require.Equal(t, http.StatusOK, regRec.Code, "Registration failed")

	loginReqBody, _ := json.Marshal(models.Credentials{Login: testUsername, Password: testPassword})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(loginReqBody))
	loginRec := httptest.NewRecorder()
	testApp.LoginUserHandler(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code, "Login failed")

	var tokenResp orchestratorApp.TokenResponse
	err = json.NewDecoder(loginRec.Body).Decode(&tokenResp)
	require.NoError(t, err, "Failed to decode login token response")
	testUserToken = tokenResp.Token
	require.NotEmpty(t, testUserToken, "Token is empty")

	claims, err := auth.ValidateToken(testUserToken)
	require.NoError(t, err, "Failed to validate test token")
	testUserID = claims.UserID

	return func() {
		testDB.Close()
		os.Unsetenv("JWT_SECRET")
	}
}

func TestOrchestratorApp_AuthHandlers(t *testing.T) {
	teardown := setupTestApp(t)
	defer teardown()

	t.Run("Register new user", func(t *testing.T) {
		creds := models.Credentials{Login: "newuser", Password: "newpassword"}
		body, _ := json.Marshal(creds)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		testApp.RegisterUserHandler(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Register existing user", func(t *testing.T) {
		creds := models.Credentials{Login: "testuser", Password: "anotherpassword"}
		body, _ := json.Marshal(creds)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		testApp.RegisterUserHandler(rr, req)
		assert.Equal(t, http.StatusConflict, rr.Code)
		var errResp orchestratorApp.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&errResp)
		assert.Contains(t, errResp.Error, "already exists")
	})

	t.Run("Login with valid credentials", func(t *testing.T) {
		creds := models.Credentials{Login: "testuser", Password: "testpassword"}
		body, _ := json.Marshal(creds)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		testApp.LoginUserHandler(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var tokenResp orchestratorApp.TokenResponse
		json.NewDecoder(rr.Body).Decode(&tokenResp)
		assert.NotEmpty(t, tokenResp.Token)
	})

	t.Run("Login with invalid password", func(t *testing.T) {
		creds := models.Credentials{Login: "testuser", Password: "wrongpassword"}
		body, _ := json.Marshal(creds)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		testApp.LoginUserHandler(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Login non-existent user", func(t *testing.T) {
		creds := models.Credentials{Login: "nosuchuser", Password: "password"}
		body, _ := json.Marshal(creds)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		testApp.LoginUserHandler(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestOrchestratorApp_CalculatorHandler(t *testing.T) {
	teardown := setupTestApp(t)
	defer teardown()

	authMiddleware := middleware.AuthMiddleware(http.HandlerFunc(testApp.CalculatorHandler))

	tests := []struct {
		name              string
		expression        string
		expectedStatus    int
		expectedTaskCount int
		expectErrorMsg    string
	}{
		{"Valid expression 3+4", "3+4", http.StatusCreated, 1, ""},
		{"Invalid RPN expression 34+", "34+", http.StatusUnprocessableEntity, 0, "Expression is not valid"},
		{"Invalid char 3&4", "3&4", http.StatusUnprocessableEntity, 0, "Expression is not valid"},
		{"Empty expression", "", http.StatusBadRequest, 0, "Expression is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testApp.TaskStore.GetTasks()

			reqBody, _ := json.Marshal(orchestratorApp.ExpressionRequest{Expression: tt.expression})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBuffer(reqBody))
			req.Header.Set("Authorization", "Bearer "+testUserToken)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			authMiddleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			if tt.expectedStatus == http.StatusCreated {
				var successResp orchestratorApp.SuccessResponse
				err := json.NewDecoder(w.Body).Decode(&successResp)
				require.NoError(t, err, "Failed to decode success response")
				assert.NotEmpty(t, successResp.Id, "Expected expression ID in response")

				expr, exists := testApp.ExpressionStore.GetExpression(successResp.Id, testUserID)
				assert.True(t, exists, "Expression not found in DB")
				assert.Equal(t, tt.expression, expr.ExpressionString)
				assert.Equal(t, "in progress", expr.Status)
				assert.Equal(t, testUserID, expr.UserID)

				assert.Len(t, testApp.TaskStore.GetTasks(), tt.expectedTaskCount, "Task count in TaskStore mismatch")

			} else {
				var errorResp orchestratorApp.ErrorResponse
				err := json.NewDecoder(w.Body).Decode(&errorResp)
				require.NoError(t, err, "Failed to decode error response")
				assert.Contains(t, errorResp.Error, tt.expectErrorMsg, "Error message mismatch")
			}
		})
	}
}

func TestOrchestratorApp_InternalTaskHandlers(t *testing.T) {
	teardown := setupTestApp(t)
	defer teardown()

	calcAuthMiddleware := middleware.AuthMiddleware(http.HandlerFunc(testApp.CalculatorHandler))
	exprReqBody, _ := json.Marshal(orchestratorApp.ExpressionRequest{Expression: "5+5"})
	calcReq := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBuffer(exprReqBody))
	calcReq.Header.Set("Authorization", "Bearer "+testUserToken)
	calcReq.Header.Set("Content-Type", "application/json")
	calcRec := httptest.NewRecorder()
	calcAuthMiddleware.ServeHTTP(calcRec, calcReq)
	require.Equal(t, http.StatusCreated, calcRec.Code)
	var successResp orchestratorApp.SuccessResponse
	json.NewDecoder(calcRec.Body).Decode(&successResp)
	expressionID := successResp.Id

	t.Run("GetInternalTaskHandler - task available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/task/new", nil)
		w := httptest.NewRecorder()
		testApp.GetInternalTaskHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var task internal.Task
		err := json.NewDecoder(w.Body).Decode(&task)
		require.NoError(t, err)
		assert.Equal(t, expressionID, task.Id)
		assert.Equal(t, "5", task.Arg1)
		assert.Equal(t, "5", task.Arg2)
		assert.Equal(t, "+", task.Operation)
	})

	t.Run("InternalTaskResultHandler - valid result", func(t *testing.T) {
		resultData := internal.TaskResult{Id: expressionID, Result: "10.0000000000"}
		body, _ := json.Marshal(resultData)

		req := httptest.NewRequest(http.MethodPost, "/internal/task", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testApp.InternalTaskResultHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		_, exists := testApp.TaskStore.TasksResStore.GetTaskRes(expressionID)
		assert.True(t, exists, "Task result not found in TaskResultStore")

		expr, exists := testApp.ExpressionStore.GetExpression(expressionID, testUserID)
		assert.True(t, exists, "Expression not found in DB after result")
		assert.Equal(t, "calculated", expr.Status)
		assert.Equal(t, "10.0000000000", expr.Result)
	})

	t.Run("GetInternalTaskHandler - no tasks available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/task/new", nil)
		w := httptest.NewRecorder()
		testApp.GetInternalTaskHandler(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestOrchestratorApp_GetExpressionsHandlers(t *testing.T) {
	teardown := setupTestApp(t)
	defer teardown()

	getExpressionsAuth := middleware.AuthMiddleware(http.HandlerFunc(testApp.GetExpressionsHandler))
	getExpressionByIdAuth := middleware.AuthMiddleware(http.HandlerFunc(testApp.GetExpressionByIdHandler))

	expr1 := internal.Expression{ID: "expr1-uuid", UserID: testUserID, ExpressionString: "1+1", Status: "calculated", Result: "2.0"}
	expr2 := internal.Expression{ID: "expr2-uuid", UserID: testUserID, ExpressionString: "2+2", Status: "in progress"}
	err := testApp.ExpressionStore.AddExpression(expr1)
	require.NoError(t, err)
	err = testApp.ExpressionStore.AddExpression(expr2)
	require.NoError(t, err)

	_, err = testApp.UserStore.CreateUser("anotheruser", "password")
	require.NoError(t, err)
	otherUser, err := testApp.UserStore.GetUserByLogin("anotheruser")
	require.NoError(t, err)
	exprOtherUser := internal.Expression{ID: "expr3-uuid", UserID: otherUser.ID, ExpressionString: "3+3", Status: "calculated", Result: "6.0"}
	err = testApp.ExpressionStore.AddExpression(exprOtherUser)
	require.NoError(t, err)

	t.Run("GetExpressionsHandler - successfully retrieves user's expressions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/expressions", nil)
		req.Header.Set("Authorization", "Bearer "+testUserToken)
		w := httptest.NewRecorder()

		getExpressionsAuth.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var expressions []internal.Expression
		err := json.NewDecoder(w.Body).Decode(&expressions)
		require.NoError(t, err)
		assert.Len(t, expressions, 2, "Should only retrieve expressions for the authenticated user")
	})

	t.Run("GetExpressionByIdHandler - successfully retrieves specific expression", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/expressions/"+expr1.ID, nil)
		req.Header.Set("Authorization", "Bearer "+testUserToken)
		w := httptest.NewRecorder()

		getExpressionByIdAuth.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var expression internal.Expression
		err := json.NewDecoder(w.Body).Decode(&expression)
		require.NoError(t, err)
		assert.Equal(t, expr1.ID, expression.ID)
		assert.Equal(t, expr1.ExpressionString, expression.ExpressionString)
	})

	t.Run("GetExpressionByIdHandler - expression not found for user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/expressions/nonexistent-id", nil)
		req.Header.Set("Authorization", "Bearer "+testUserToken)
		w := httptest.NewRecorder()
		getExpressionByIdAuth.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetExpressionByIdHandler - expression belongs to another user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/expressions/"+exprOtherUser.ID, nil)
		req.Header.Set("Authorization", "Bearer "+testUserToken)
		w := httptest.NewRecorder()
		getExpressionByIdAuth.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestOrchestratorApp_getTimeSetting(t *testing.T) {
	os.Setenv("TIME_ADDITION_MS", "101")
	os.Setenv("TIME_SUBTRACTION_MS", "102")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "203")
	os.Setenv("TIME_DIVISIONS_MS", "204")
	defer func() {
		os.Unsetenv("TIME_ADDITION_MS")
		os.Unsetenv("TIME_SUBTRACTION_MS")
		os.Unsetenv("TIME_MULTIPLICATIONS_MS")
		os.Unsetenv("TIME_DIVISIONS_MS")
	}()

	tests := []struct {
		operation string
		expected  string
	}{
		{"+", "101"},
		{"-", "102"},
		{"*", "203"},
		{"/", "204"},
		{"unknown", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			t.Skip("getTimeSetting is unexported. Test it indirectly or export it.")
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	teardown := setupTestApp(t)
	defer teardown()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
		require.True(t, ok, "UserID not found in context or not int64")
		assert.Equal(t, testUserID, userID, "UserID in context does not match expected")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	authHandler := middleware.AuthMiddleware(nextHandler)

	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+testUserToken)
		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "OK", rr.Body.String())
	})

	t.Run("No token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var errResp map[string]string
		json.NewDecoder(rr.Body).Decode(&errResp)
		assert.Contains(t, errResp["error"], "missing")
	})

	t.Run("Malformed token header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bear testUserToken")
		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var errResp map[string]string
		json.NewDecoder(rr.Body).Decode(&errResp)
		assert.Contains(t, errResp["error"], "format")
	})

	t.Run("Invalid token (bad signature or expired)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+"invalidtokenstring")
		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var errResp map[string]string
		json.NewDecoder(rr.Body).Decode(&errResp)
		assert.Contains(t, errResp["error"], "Invalid token")
	})
}

func TestTaskStore_GetFirstCorrectTask(t *testing.T) {
	ts := store.NewTaskStore()

	t.Run("No tasks", func(t *testing.T) {
		_, exists := ts.GetFirstCorrectTask()
		assert.False(t, exists)
	})

	task1 := internal.Task{Id: "t1", Arg1: "2", Arg2: "3", Operation: "+", Operation_time: "10"}
	ts.AddTask(task1)

	t.Run("Simple numeric task", func(t *testing.T) {
		task, exists := ts.GetFirstCorrectTask()
		assert.True(t, exists)
		assert.Equal(t, task1.Id, task.Id)
		assert.Equal(t, "2", task.Arg1)
		_, exists = ts.GetFirstCorrectTask()
		assert.False(t, exists, "Task should have been removed")
	})

	ts = store.NewTaskStore()
	taskDep1 := internal.Task{Id: "td1", Arg1: "resA", Arg2: "5", Operation: "*", Operation_time: "10"}
	taskDep2 := internal.Task{Id: "td2", Arg1: "10", Arg2: "resB", Operation: "-", Operation_time: "10"}
	taskReady := internal.Task{Id: "tr1", Arg1: "20", Arg2: "2", Operation: "/", Operation_time: "10"}
	ts.AddTask(taskDep1)
	ts.AddTask(taskDep2)
	ts.AddTask(taskReady)

	t.Run("One ready task among dependent tasks", func(t *testing.T) {
		task, exists := ts.GetFirstCorrectTask()
		assert.True(t, exists)
		assert.Equal(t, taskReady.Id, task.Id)
		tasksLeft := ts.GetTasks()
		assert.Len(t, tasksLeft, 2)
	})

	ts.TasksResStore.AddTaskRes(internal.TaskResult{Id: "resA", Result: "4.0"})

	t.Run("Task becomes ready after dependency resolved", func(t *testing.T) {
		task, exists := ts.GetFirstCorrectTask()
		assert.True(t, exists, "TaskDep1 should be ready")
		assert.Equal(t, taskDep1.Id, task.Id)
		assert.Equal(t, "4.0", task.Arg1) // Check resolved arg
		assert.Equal(t, "5", task.Arg2)
		tasksLeft := ts.GetTasks()
		assert.Len(t, tasksLeft, 1)
		assert.Equal(t, taskDep2.Id, tasksLeft[0].Id)
	})

	ts.TasksResStore.AddTaskRes(internal.TaskResult{Id: "resB", Result: "3.0"})

	t.Run("Last task becomes ready", func(t *testing.T) {
		task, exists := ts.GetFirstCorrectTask()
		assert.True(t, exists, "TaskDep2 should be ready")
		assert.Equal(t, taskDep2.Id, task.Id)
		assert.Equal(t, "10", task.Arg1)
		assert.Equal(t, "3.0", task.Arg2)
		_, exists = ts.GetFirstCorrectTask()
		assert.False(t, exists)
	})
}
