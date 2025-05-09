package store

import (
	"strconv"
	"sync"

	"github.com/katierevinska/calculatorService/internal"
)

type Counter struct {
	value int
	mu    sync.RWMutex
}

func (c *Counter) GetValueAndInc() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
	return c.value
}

func NewCounter() *Counter {
	return &Counter{
		value: 0,
	}
}

type TaskResultStore struct {
	tasksRes map[string]internal.TaskResult
	mu       sync.Mutex
}

func NewTaskResultStore() *TaskResultStore {
	return &TaskResultStore{
		tasksRes: make(map[string]internal.TaskResult),
	}
}

func (store *TaskResultStore) AddTaskRes(t internal.TaskResult) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.tasksRes[t.Id] = t
}

func (store *TaskResultStore) GetTaskRes(id string) (internal.TaskResult, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	task, exists := store.tasksRes[id]
	return task, exists
}

type TaskStore struct {
	tasks         []internal.Task
	TasksResStore TaskResultStore
	Counter       Counter
	mu            sync.Mutex
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		TasksResStore: *NewTaskResultStore(),
		tasks:         []internal.Task{},
		Counter:       *NewCounter(),
	}
}

func (store *TaskStore) AddTask(t internal.Task) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.tasks = append(store.tasks, t)
}

func (store *TaskStore) GetTasks() []internal.Task {
	store.mu.Lock()
	defer store.mu.Unlock()
	tasksCopy := make([]internal.Task, len(store.tasks))
	copy(tasksCopy, store.tasks)
	return tasksCopy
}

func (store *TaskStore) GetFirstCorrectTask() (internal.Task, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if len(store.tasks) == 0 {
		return internal.Task{}, false
	}

	for i, task := range store.tasks {
		resolvedArg1 := task.Arg1
		isArg1Numeric := false
		if _, err := strconv.ParseFloat(task.Arg1, 64); err == nil {
			isArg1Numeric = true
		} else {
			if res, exists := store.TasksResStore.GetTaskRes(task.Arg1); exists {
				resolvedArg1 = res.Result
				isArg1Numeric = true
			}
		}

		resolvedArg2 := task.Arg2
		isArg2Numeric := false
		if _, err := strconv.ParseFloat(task.Arg2, 64); err == nil {
			isArg2Numeric = true
		} else {
			if res, exists := store.TasksResStore.GetTaskRes(task.Arg2); exists {
				resolvedArg2 = res.Result
				isArg2Numeric = true
			}
		}

		if isArg1Numeric && isArg2Numeric {
			readyTask := task
			readyTask.Arg1 = resolvedArg1
			readyTask.Arg2 = resolvedArg2

			store.tasks = append(store.tasks[:i], store.tasks[i+1:]...)
			return readyTask, true
		}
	}
	return internal.Task{}, false
}
