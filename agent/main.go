package main

import (
	"os"

	AgentApp "github.com/katierevinska/calculatorService/internal/applications/agent_app"
)

func main() {
	SetEnvVariables()
	app := AgentApp.New()
	app.RunServer()
}

func SetEnvVariables() {
	os.Setenv("COMPUTING_POWER", "20")
}
