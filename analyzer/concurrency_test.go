package analyzer

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkerPool_AsyncProcessingAndShutdown(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 3, 10)
	service.Start()

	inputs := []string{
		"Hello Go concurrency",
		"Worker pool pattern",
		"Channels and mutexes",
		"Parallel execution test",
		"Fifth independent task",
	}

	var taskIDs []int
	for _, text := range inputs {
		task, err := service.CreateTask(text)
		if err != nil {
			t.Fatalf("ошибка создания задачи: %v", err)
		}
		if err := service.SubmitTask(task.ID); err != nil {
			t.Fatalf("ошибка отправки задачи #%d в очередь: %v", task.ID, err)
		}
		taskIDs = append(taskIDs, task.ID)
	}

	service.Shutdown()
	service.Shutdown()

	for _, id := range taskIDs {
		task, err := service.GetTask(id)
		if err != nil {
			t.Fatalf("не удалось получить задачу #%d: %v", id, err)
		}
		if task.State != StateCompleted {
			t.Errorf("задача #%d должна быть в статусе %s, получено %s", id, StateCompleted, task.State)
		}
		if task.Result == nil || task.Result.WordCount != 3 {
			t.Errorf("некорректный результат у задачи #%d: %+v", id, task.Result)
		}
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 2, 5)

	var activeWorkers atomic.Int32
	twoWorkersActive := make(chan struct{})
	releaseWorkers := make(chan struct{})
	var once sync.Once

	service.processHook = func(taskID int) {
		current := activeWorkers.Add(1)
		if current >= 2 {
			once.Do(func() {
				close(twoWorkersActive)
			})
		}
		<-releaseWorkers
		activeWorkers.Add(-1)
	}

	service.Start()

	t1, _ := service.CreateTask("First concurrent task")
	t2, _ := service.CreateTask("Second concurrent task")

	if err := service.SubmitTask(t1.ID); err != nil {
		t.Fatalf("не удалось отправить t1: %v", err)
	}
	if err := service.SubmitTask(t2.ID); err != nil {
		t.Fatalf("не удалось отправить t2: %v", err)
	}

	<-twoWorkersActive
	close(releaseWorkers)

	service.Shutdown()
}

func TestWorkerPool_InvalidOrDuplicateSubmission(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 2, 5)
	service.Start()
	defer service.Shutdown()

	task, _ := service.CreateTask("Duplicate test")

	if err := service.SubmitTask(task.ID); err != nil {
		t.Fatalf("первая отправка должна пройти успешно: %v", err)
	}

	err := service.SubmitTask(task.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("ожидалась ошибка %v при дублирующей отправке, получена %v", ErrInvalidTransition, err)
	}

	err = service.SubmitTask(9999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("ожидалась ошибка %v для несуществующей задачи, получена %v", ErrTaskNotFound, err)
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 2, 2)

	blockWorkers := make(chan struct{})
	workersStarted := make(chan struct{}, 2)

	service.processHook = func(taskID int) {
		workersStarted <- struct{}{}
		<-blockWorkers
	}

	service.Start()

	tasks := make([]*Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i], _ = service.CreateTask(fmt.Sprintf("Task number %d", i))
	}

	_ = service.SubmitTask(tasks[0].ID)
	_ = service.SubmitTask(tasks[1].ID)
	<-workersStarted
	<-workersStarted

	if err := service.SubmitTask(tasks[2].ID); err != nil {
		t.Fatalf("задача 2 должна поместиться в буфер очереди: %v", err)
	}
	if err := service.SubmitTask(tasks[3].ID); err != nil {
		t.Fatalf("задача 3 должна поместиться в буфер очереди: %v", err)
	}

	err := service.SubmitTask(tasks[4].ID)
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("ожидалась ошибка %v при переполнении очереди, получена %v", ErrQueueFull, err)
	}

	rejectedTask, _ := service.GetTask(tasks[4].ID)
	if rejectedTask.State != StatePending {
		t.Errorf("отклоненная задача должна остаться в статусе %s, получено %s", StatePending, rejectedTask.State)
	}

	close(blockWorkers)
	service.Shutdown()
}

func TestWorkerPool_ConcurrentAccessSafety(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 4, 50)
	service.Start()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			task, err := service.CreateTask(fmt.Sprintf("Concurrent message %d", idx))
			if err != nil {
				t.Errorf("ошибка создания: %v", err)
				return
			}
			_ = service.SubmitTask(task.ID)
			_, _ = service.GetTask(task.ID)
		}(i)
	}

	wg.Wait()
	service.Shutdown()
}

func TestWorkerPool_RejectAfterShutdown(t *testing.T) {
	service := NewServiceWithConfig(NewInMemoryStorage(), 2, 5)
	service.Start()

	task, _ := service.CreateTask("Task after stop")
	service.Shutdown()

	err := service.SubmitTask(task.ID)
	if !errors.Is(err, ErrServiceStopped) {
		t.Errorf("ожидалась ошибка %v после остановки сервиса, получена %v", ErrServiceStopped, err)
	}
}
