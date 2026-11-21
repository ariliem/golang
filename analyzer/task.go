package analyzer

import "errors"

type State string

const (
	StatePending    State = "Pending"
	StateQueued     State = "Queued"
	StateProcessing State = "Processing"
	StateCompleted  State = "Completed"
)

var (
	ErrEmptyInput        = errors.New("текст не может быть пустым или не должны быть только пробелы")
	ErrTaskNotFound      = errors.New("задачи с такой айдишкой нет")
	ErrInvalidTransition = errors.New("недопустимый переход состояния или повторная обработка")

	ErrQueueFull         = errors.New("очередь задач переполнена")
	ErrServiceStopped    = errors.New("пул воркеров остановлен, новые задачи не принимаются")
	ErrServiceNotStarted = errors.New("пул воркеров еще не запущен")
)

type Result struct {
	WordCount   int
	CharCount   int
	VowelCount  int
	LongestWord string
}

type Task struct {
	ID     int
	Input  string
	State  State
	Result *Result
}

func (t Task) CanProcess() bool {
	return t.State == StatePending
}

func (t Task) CanSubmit() bool {
	return t.State == StatePending
}

func (t *Task) TransitionTo(newState State) error {
	switch t.State {
	case StatePending:

		if newState == StateQueued || newState == StateProcessing {
			t.State = newState
			return nil
		}
	case StateQueued:

		if newState == StateProcessing || newState == StatePending {
			t.State = newState
			return nil
		}
	case StateProcessing:
		if newState == StateCompleted {
			t.State = newState
			return nil
		}
	}
	return ErrInvalidTransition
}
