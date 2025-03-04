package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/katierevinska/calculatorService/internal"
)

type AgentApp struct {
}

func New() *AgentApp {
	return &AgentApp{}
}

func worker(id int, tasks <-chan internal.Task, results chan<- internal.TaskResult) {
	//пусть агент добавляет в jobs,
	//тогда тут ниже будут перебираться задачи оттуда
	//только должны быть не числа а инстансы
	//и посчитав в канал результатов
	for t := range tasks {
		fmt.Println(id)
		opTime, _ := strconv.ParseInt(t.Operation_time, 10, 64)
		time.Sleep(time.Duration(opTime))
		resultValue := calculate(t)
		log.Println("calculate a value of task and result is " + resultValue)
		results <- internal.TaskResult{Id: t.Id, Result: resultValue}
	}
}

func calculate(t internal.Task) string {
	var result float64
	a, errA := strconv.ParseFloat(t.Arg1, 64)
	b, errB := strconv.ParseFloat(t.Arg2, 64)

	if errA != nil || errB != nil {
		return "Error: Invalid number"
	}

	switch t.Operation {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		result = a / b
	}
	return strconv.FormatFloat(result, 'f', 10, 64)
}

func (a *AgentApp) RunServer() {
	tasks := make(chan internal.Task, 100)
	results := make(chan internal.TaskResult, 100)
	num, _ := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	for w := 1; w <= num; w++ {
		go worker(w, tasks, results)
	}
	go func() {
		for result := range results {
			log.Println("try to send to orchestrator " + result.Result)
			sendResult(result)
		}
	}()
	//здесь постоянно опрашивает оркестратора есть ли работа
	//и добавляем в канал полученные задачи
	for {
		task := fetchTask()
		if task != nil {
			tasks <- *task
		} else {
			time.Sleep(time.Second * 20)
		}
	}
}

func sendResult(result internal.TaskResult) {
	log.Println("want to send to orchestrator " + result.Id + " " + result.Result)
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Ошибка при маршализации результата: %v", err)
		return
	}
	resp, err := http.Post("http://localhost:8080/internal/task", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Ошибка при отправке результата: %v", err)
		return
	}
	resp.Body.Close()
}

func fetchTask() *internal.Task {
	resp, err := http.Get("http://localhost:8080/internal/task/new")
	if err != nil {
		log.Printf("Ошибка при получении задачи: %v", err)
		time.Sleep(5 * time.Second) // Подождать перед новой попыткой
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ошибка: статус ответа %s", resp.Status)
		time.Sleep(20 * time.Second) // Подождать перед новой попыткой
		return nil
	}

	var task internal.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		log.Printf("Ошибка при декодировании задачи: %v", err)
		return nil
	}

	return &task
}
