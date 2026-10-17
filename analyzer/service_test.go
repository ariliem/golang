package analyzer

import (
	"errors"
	"testing"
)

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError error
	}{
		{
			name:      "Успешное создание задачи",
			input:     "Hello Go world",
			wantError: nil,
		},
		{
			name:      "Пустая строка",
			input:     "",
			wantError: ErrEmptyInput,
		},
		{
			name:      "Строка только из пробелов и табуляции",
			input:     "   \t\n  ",
			wantError: ErrEmptyInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(NewInMemoryStorage())
			task, err := service.CreateTask(tt.input)

			if !errors.Is(err, tt.wantError) {
				t.Fatalf("ожидалась ошибка %v, получена %v", tt.wantError, err)
			}

			if tt.wantError == nil {
				if task == nil {
					t.Fatal("ожидалась созданная задача, получен nil")
				}
				if task.State != StatePending {
					t.Errorf("ожидался статус %s, получен %s", StatePending, task.State)
				}
				if task.Result != nil {
					t.Error("до обработки результат должен быть nil")
				}
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	service := NewService(NewInMemoryStorage())
	created, _ := service.CreateTask("Sample text")

	fetched, err := service.GetTask(created.ID)
	if err != nil {
		t.Fatalf("неожиданная ошибка при получении задачи: %v", err)
	}
	if fetched.ID != created.ID || fetched.Input != "Sample text" {
		t.Errorf("данные полученной задачи не совпадают с созданной")
	}

	_, err = service.GetTask(999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("ожидалась ошибка %v, получена %v", ErrTaskNotFound, err)
	}
}

func TestProcessTask_Success(t *testing.T) {
	service := NewService(NewInMemoryStorage())
	task, _ := service.CreateTask("Hello Go world")

	processed, err := service.ProcessTask(task.ID)
	if err != nil {
		t.Fatalf("ошибка при обработке: %v", err)
	}

	if processed.State != StateCompleted {
		t.Errorf("ожидался статус %s, получен %s", StateCompleted, processed.State)
	}

	if processed.Result == nil {
		t.Fatal("ожидался результат, получен nil")
	}
	if processed.Result.WordCount != 3 {
		t.Errorf("ожидалось 3 слова, получено %d", processed.Result.WordCount)
	}
	if processed.Result.CharCount != 14 {
		t.Errorf("ожидалось 14 символов, получено %d", processed.Result.CharCount)
	}
	if processed.Result.VowelCount != 4 { // e, o, o, o
		t.Errorf("ожидалось 4 гласных, получено %d", processed.Result.VowelCount)
	}
	if processed.Result.LongestWord != "Hello" {
		t.Errorf("ожидалось самое длинное слово 'Hello', получено '%s'", processed.Result.LongestWord)
	}
}

func TestProcessTask_InvalidTransition(t *testing.T) {
	service := NewService(NewInMemoryStorage())
	task, _ := service.CreateTask("Test text")

	_, err := service.ProcessTask(task.ID)
	if err != nil {
		t.Fatalf("первая обработка не должна возвращать ошибку: %v", err)
	}

	_, err = service.ProcessTask(task.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("ожидалась ошибка %v при повторной обработке, получена %v", ErrInvalidTransition, err)
	}
}

func TestMultipleIndependentTasks(t *testing.T) {
	service := NewService(NewInMemoryStorage())

	t1, _ := service.CreateTask("Short")
	t2, _ := service.CreateTask("A much longer sentence here")

	if t1.ID == t2.ID {
		t.Fatal("ID разных задач должны быть уникальными")
	}

	_, err := service.ProcessTask(t1.ID)
	if err != nil {
		t.Fatalf("ошибка при обработке первой задачи: %v", err)
	}

	fetched1, _ := service.GetTask(t1.ID)
	fetched2, _ := service.GetTask(t2.ID)

	if fetched1.State != StateCompleted || fetched1.Result.WordCount != 1 {
		t.Errorf("некорректное состояние или результат первой задачи")
	}
	if fetched2.State != StatePending || fetched2.Result != nil {
		t.Errorf("вторая задача не должна была измениться")
	}
}
