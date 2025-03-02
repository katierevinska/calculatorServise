package application

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type AgentApp struct {
}

func New() *AgentApp {
	return &AgentApp{}
}

func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		//ask for task + sleep how needed
		time.Sleep(time.Second)
		fmt.Println("worker", id, "finished job", j)
		results <- j * 2
	}
}

func (a *AgentApp) RunServer() {
	jobs := make(chan int, 100)
	results := make(chan int, 100)
	num, _ := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	for w := 1; w <= num; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= 5; a++ {
		<-results
	}
}
