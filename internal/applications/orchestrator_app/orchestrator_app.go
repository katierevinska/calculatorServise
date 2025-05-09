package orchestrator_app // Убедись, что имя пакета совпадает с директорией

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/internal/auth"
	"github.com/katierevinska/calculatorService/internal/middleware"
	"github.com/katierevinska/calculatorService/internal/models"
	"github.com/katierevinska/calculatorService/internal/store"
	"github.com/katierevinska/calculatorService/pkg/rpn"
)

type OrchestratorApp struct {
	db              *sql.DB
	UserStore       *store.UserStore
	ExpressionStore *store.ExpressionStore
	TaskStore       *store.TaskStore
}

func New(db *sql.DB) *OrchestratorApp {
	return &OrchestratorApp{
		db:              db,
		UserStore:       store.NewUserStore(db),
		ExpressionStore: store.NewExpressionStore(db),
		TaskStore:       store.NewTaskStore(),
	}
}

func (app *OrchestratorApp) RunServer() {
	http.HandleFunc("/api/v1/register", app.RegisterUserHandler)
	http.HandleFunc("/api/v1/login", app.LoginUserHandler)

	calculateHandler := http.HandlerFunc(app.CalculatorHandler)
	expressionsHandler := http.HandlerFunc(app.GetExpressionsHandler)
	expressionByIdHandler := http.HandlerFunc(app.GetExpressionByIdHandler)

	http.Handle("/api/v1/calculate", middleware.AuthMiddleware(calculateHandler))
	http.Handle("/api/v1/expressions", middleware.AuthMiddleware(expressionsHandler))
	http.Handle("/api/v1/expressions/", middleware.AuthMiddleware(expressionByIdHandler))

	http.HandleFunc("/internal/task/new", app.GetInternalTaskHandler)
	http.HandleFunc("/internal/task", app.InternalTaskResultHandler)

	log.Println("Orchestrator server starting on :8080")
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
type TokenResponse struct {
	Token string `json:"token"`
}

func (app *OrchestratorApp) GetExpressionByIdHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		log.Println("GetExpressionByIdHandler: Failed to get userID from context")
		app.jsonErrorResponse(w, "Internal server error (userID missing in context)", http.StatusInternalServerError)
		return
	}

	idStr := r.URL.Path[len("/api/v1/expressions/"):]
	if idStr == "" {
		app.jsonErrorResponse(w, "Expression ID is missing in path", http.StatusBadRequest)
		return
	}

	expression, exists := app.ExpressionStore.GetExpression(idStr, userID)
	if !exists {
		log.Printf("Expression %s not found for user %d", idStr, userID)
		app.jsonErrorResponse(w, "Expression not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expression)
}

func (app *OrchestratorApp) GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		log.Println("GetExpressionsHandler: Failed to get userID from context")
		app.jsonErrorResponse(w, "Internal server error (userID missing in context)", http.StatusInternalServerError)
		return
	}

	exps := app.ExpressionStore.GetAllExpressions(userID)
	log.Printf("Path: %s - send all expressions for user %d, found %d", r.URL.Path, userID, len(exps))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(exps)
}

func (app *OrchestratorApp) getTimeSetting(operation string) string {
	switch operation {
	case "+":
		return os.Getenv("TIME_ADDITION_MS")
	case "-":
		return os.Getenv("TIME_SUBTRACTION_MS")
	case "*":
		return os.Getenv("TIME_MULTIPLICATIONS_MS")
	case "/":
		return os.Getenv("TIME_DIVISIONS_MS")
	default:
		log.Printf("Warning: Unknown operation '%s' requested for time setting, returning 0ms", operation)
		return "0"
	}
}
func (app *OrchestratorApp) CalculatorHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		log.Println("CalculatorHandler: Failed to get userID from context")
		app.jsonErrorResponse(w, "Internal server error (userID missing in context)", http.StatusInternalServerError)
		return
	}

	var requestExrp ExpressionRequest
	if err := json.NewDecoder(r.Body).Decode(&requestExrp); err != nil {
		app.jsonErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if requestExrp.Expression == "" {
		app.jsonErrorResponse(w, "Expression is empty", http.StatusBadRequest)
		return
	}

	expressionID, err := rpn.Calc(requestExrp.Expression, app.TaskStore)
	if err != nil {
		log.Printf("Error from rpn.Calc for expression '%s' by user %d: %v", requestExrp.Expression, userID, err)
		app.jsonErrorResponse(w, "Expression is not valid or processing error: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	newExpr := internal.Expression{
		ID:               expressionID,
		UserID:           userID,
		ExpressionString: requestExrp.Expression,
		Status:           "in progress",
		Result:           "",
	}
	if err := app.ExpressionStore.AddExpression(newExpr); err != nil {
		log.Printf("Failed to add expression %s to store for user %d: %v", expressionID, userID, err)
		app.jsonErrorResponse(w, "Failed to save expression", http.StatusInternalServerError)
		return
	}

	log.Printf("Expression '%s' (ID: %s) accepted from user %d", requestExrp.Expression, expressionID, userID)
	res := SuccessResponse{Id: expressionID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (app *OrchestratorApp) InternalTaskResultHandler(w http.ResponseWriter, r *http.Request) {
	var resultData internal.TaskResult

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&resultData); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	log.Printf("Received task result from agent: ID %s, Result %s", resultData.Id, resultData.Result)
	app.TaskStore.TasksResStore.AddTaskRes(resultData)

	err := app.ExpressionStore.UpdateExpressionStatusResult(resultData.Id, "calculated", resultData.Result)
	if err != nil {
		log.Printf("Could not update expression status for ID %s, perhaps it's an intermediate task or DB error.", resultData.Id)
	} else {
		log.Printf("Expression with ID %s (assumed final task) updated to 'calculated' with result '%s'", resultData.Id, resultData.Result)
	}

	w.WriteHeader(http.StatusOK)
}

func (app *OrchestratorApp) GetInternalTaskHandler(w http.ResponseWriter, r *http.Request) {
	task, exists := app.TaskStore.GetFirstCorrectTask()
	if exists {
		log.Println("Agent asked for task, sending task ID: " + task.Id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(task)
		return
	}
	log.Println("Agent asked for task but no ready tasks available.")
	w.WriteHeader(http.StatusNotFound)
}

func (app *OrchestratorApp) jsonErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func (app *OrchestratorApp) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		app.jsonErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if creds.Login == "" || creds.Password == "" {
		app.jsonErrorResponse(w, "Login and password are required", http.StatusBadRequest)
		return
	}

	_, err := app.UserStore.CreateUser(creds.Login, creds.Password)
	if err != nil {
		if errors.Is(err, store.ErrUserExists) {
			app.jsonErrorResponse(w, "User with this login already exists", http.StatusConflict)
		} else {
			log.Printf("Error creating user '%s': %v", creds.Login, err)
			app.jsonErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("User '%s' registered successfully", creds.Login)
	w.WriteHeader(http.StatusOK)
}

func (app *OrchestratorApp) LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		app.jsonErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	user, err := app.UserStore.GetUserByLogin(creds.Login)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			app.jsonErrorResponse(w, "Invalid login or password", http.StatusUnauthorized)
		} else {
			log.Printf("Error fetching user '%s' for login: %v", creds.Login, err)
			app.jsonErrorResponse(w, "Login failed", http.StatusInternalServerError)
		}
		return
	}

	if !models.CheckPasswordHash(creds.Password, user.PasswordHash) {
		app.jsonErrorResponse(w, "Invalid login or password", http.StatusUnauthorized)
		return
	}

	tokenString, err := auth.GenerateToken(user.ID)
	if err != nil {
		log.Printf("Error generating token for user '%s': %v", creds.Login, err)
		app.jsonErrorResponse(w, "Login failed (token generation)", http.StatusInternalServerError)
		return
	}

	log.Printf("User '%s' (ID: %d) logged in successfully", creds.Login, user.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TokenResponse{Token: tokenString})
}
