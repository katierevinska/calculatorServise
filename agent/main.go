package main

import (
	"os"

	AgentApp "github.com/katierevinska/rpn/internal/applications/agent_app"
)

func main() {
	SetEnvVariables()
	app := AgentApp.New()
	app.RunServer()
}

func SetEnvVariables() {
	os.Setenv("COMPUTING_POWER", "200")
}
