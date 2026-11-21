package analyzer

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultWorkerCount   = 3
	DefaultQueueCapacity = 10
)

type Service struct {
	repo   TaskRepository
	nextID int

	workerCount int
	queue       chan int

	mu      sync.Mutex
	stateMu sync.Mutex
	wg      sync.WaitGroup

	started bool
	stopped bool

	processHook func(taskID int)
}

func NewService(repo TaskRepository) *Service {
	return NewServiceWithConfig(repo, DefaultWorkerCount, DefaultQueueCapacity)
}

func NewServiceWithConfig(repo TaskRepository, workerCount, queueCapacity int) *Service {
	if workerCount < 2 {
		workerCount = 2
	}
	if queueCapacity < 1 {
		queueCapacity = 1
	}
	return &Service{
		repo:        repo,
		nextID:      1,
		workerCount: workerCount,
		queue:       make(chan int, queueCapacity),
	}
}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started || s.stopped {
		return
	}
	s.started = true

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

func (s *Service) worker() {
	defer s.wg.Done()

	for taskID := range s.queue {
		s.executeTask(taskID)
	}
}

func (s *Service) CreateTask(input string) (*Task, error) {
	if strings.TrimSpace(input) == "" {
		return nil, ErrEmptyInput
	}

	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.mu.Unlock()

	task := &Task{
		ID:    id,
		Input: input,
		State: StatePending,
	}

	s.repo.Save(task)
	return task, nil
}

func (s *Service) GetTask(id int) (*Task, error) {
	return s.repo.GetByID(id)
}

func (s *Service) SubmitTask(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return ErrServiceNotStarted
	}
	if s.stopped {
		return ErrServiceStopped
	}

	s.stateMu.Lock()
	task, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return err
	}
	if !task.CanSubmit() {
		s.stateMu.Unlock()
		return ErrInvalidTransition
	}
	if err := task.TransitionTo(StateQueued); err != nil {
		s.stateMu.Unlock()
		return err
	}
	s.repo.Save(task)
	s.stateMu.Unlock()

	select {
	case s.queue <- id:
		return nil
	default:

		s.stateMu.Lock()
		_ = task.TransitionTo(StatePending)
		s.repo.Save(task)
		s.stateMu.Unlock()
		return ErrQueueFull
	}
}

func (s *Service) ProcessTask(id int) (*Task, error) {
	s.stateMu.Lock()
	task, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return nil, err
	}

	if !task.CanProcess() {
		s.stateMu.Unlock()
		return nil, ErrInvalidTransition
	}

	if err := task.TransitionTo(StateProcessing); err != nil {
		s.stateMu.Unlock()
		return nil, err
	}
	s.repo.Save(task)
	s.stateMu.Unlock()

	result := analyzeText(task.Input)

	s.stateMu.Lock()
	task.Result = &result
	_ = task.TransitionTo(StateCompleted)
	s.repo.Save(task)
	s.stateMu.Unlock()

	return task, nil
}

func (s *Service) executeTask(id int) {
	s.stateMu.Lock()
	task, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return
	}
	if err := task.TransitionTo(StateProcessing); err != nil {
		s.stateMu.Unlock()
		return
	}
	s.repo.Save(task)
	s.stateMu.Unlock()

	if s.processHook != nil {
		s.processHook(id)
	}

	result := analyzeText(task.Input)

	s.stateMu.Lock()
	task.Result = &result
	_ = task.TransitionTo(StateCompleted)
	s.repo.Save(task)
	s.stateMu.Unlock()
}

func (s *Service) Shutdown() {
	s.mu.Lock()
	if !s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.queue)
	s.mu.Unlock()

	s.wg.Wait()
}

func analyzeText(text string) Result {
	charCount := utf8.RuneCountInString(text)
	words := strings.Fields(text)
	wordCount := len(words)

	longestWord := ""
	for _, w := range words {
		cleanWord := strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if utf8.RuneCountInString(cleanWord) > utf8.RuneCountInString(longestWord) {
			longestWord = cleanWord
		}
	}

	vowelCount := 0
	vowels := "aeiouyаеёиоуыэюя"
	for _, r := range strings.ToLower(text) {
		if strings.ContainsRune(vowels, r) {
			vowelCount++
		}
	}

	return Result{
		WordCount:   wordCount,
		CharCount:   charCount,
		VowelCount:  vowelCount,
		LongestWord: longestWord,
	}
}
