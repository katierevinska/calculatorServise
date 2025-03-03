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
		time.Sleep(time.Second)
		resultValue := calculate(t)
		results <- internal.TaskResult{Id: t.Id, Result: resultValue}
		fmt.Println("worker", id, "finished job", t)
	}
}

func calculate(t internal.Task) string {

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
			sendResult(result)
		}
	}()
	//здесь постоянно опрашивает оркестратора есть ли работа
	//и добавляем в канал полученные задачи
	for {
		task := fetchTask()
		if task != nil {
			tasks <- *task
		}
	}
}

func sendResult(result internal.TaskResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Ошибка при маршализации результата: %v", err)
		return
	}

	resp, err := http.Post(serverURL+"/internal/task", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Ошибка при отправке результата: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Результат отправлен: %s\n", result.Value)
}

func fetchTask() *internal.Task {
	resp, err := http.Get(serverURL + "/internal/task")
	if err != nil {
		log.Printf("Ошибка при получении задачи: %v", err)
		time.Sleep(5 * time.Second) // Подождать перед новой попыткой
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ошибка: статус ответа %s", resp.Status)
		return nil
	}

	var task internal.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		log.Printf("Ошибка при декодировании задачи: %v", err)
		return nil
	}

	return &task
}
