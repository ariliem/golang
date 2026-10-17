package analyzer

type TaskRepository interface {
	Save(task *Task)
	GetByID(id int) (*Task, error)
}

type InMemoryStorage struct {
	tasks map[int]*Task
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		tasks: make(map[int]*Task),
	}
}

func (s *InMemoryStorage) Save(task *Task) {
	taskCopy := *task
	if task.Result != nil {
		resCopy := *task.Result
		taskCopy.Result = &resCopy
	}
	s.tasks[task.ID] = &taskCopy
}

func (s *InMemoryStorage) GetByID(id int) (*Task, error) {
	task, exists := s.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	taskCopy := *task
	if task.Result != nil {
		resCopy := *task.Result
		taskCopy.Result = &resCopy
	}
	return &taskCopy, nil
}
