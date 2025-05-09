package main

import (
	"log"
	"os"

	orchestratorApp "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
	"github.com/katierevinska/calculatorService/internal/auth"
	"github.com/katierevinska/calculatorService/internal/database"
)

func main() {
	SetEnvVariables()

	if err := auth.InitJWT(); err != nil {
		log.Fatalf("Failed to initialize JWT: %v", err)
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/orchestrator.db"
		log.Printf("DATABASE_PATH not set, using default: %s", dbPath)
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	app := orchestratorApp.New(db)
	app.RunServer()
}

func SetEnvVariables() {
	os.Setenv("TIME_ADDITION_MS", "100")
	os.Setenv("TIME_SUBTRACTION_MS", "100")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "200")
	os.Setenv("TIME_DIVISIONS_MS", "200")
	os.Setenv("DATABASE_PATH", "./data/orchestrator.db")
	os.Setenv("JWT_SECRET", "124424-231Swsws-TDedDf")
}
