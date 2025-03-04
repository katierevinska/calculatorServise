package main

import (
	"os"

	orchestratorApp "github.com/katierevinska/calculatorService/internal/applications/orchestrator_app"
)

func main() {
	SetEnvVariables()
	app := orchestratorApp.New()
	app.RunServer()
}

func SetEnvVariables() {
	os.Setenv("TIME_ADDITION_MS", "1000")
	os.Setenv("TIME_SUBTRACTION_MS", "1000")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "2000")
	os.Setenv("TIME_DIVISIONS_MS", "2000")
}
