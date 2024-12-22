package main

import (
	"github.com/katierevinska/rpn/internal/application"
)

func main() {
	app := application.New()
	app.RunServer()
}
