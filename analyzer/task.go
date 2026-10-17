package analyzer

import "errors"

type State string

const (
	StatePending    State = "Pending"
	StateProcessing State = "Processing"
	StateCompleted  State = "Completed"
)

var (
	ErrEmptyInput        = errors.New("текст не может быть пустым или не должны быть только пробелы")
	ErrTaskNotFound      = errors.New("задачи с такой айдишкой нет")
	ErrInvalidTransition = errors.New("недопустимый переход состояния или повторная обработка")
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

func (t *Task) TransitionTo(newState State) error {
	switch t.State {
	case StatePending:
		if newState == StateProcessing {
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
