package analyzer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Service struct {
	repo   TaskRepository
	nextID int
}

func NewService(repo TaskRepository) *Service {
	return &Service{
		repo:   repo,
		nextID: 1,
	}
}

func (s *Service) CreateTask(input string) (*Task, error) {
	if strings.TrimSpace(input) == "" {
		return nil, ErrEmptyInput
	}

	task := &Task{
		ID:    s.nextID,
		Input: input,
		State: StatePending,
	}
	s.nextID++

	s.repo.Save(task)
	return task, nil
}

func (s *Service) GetTask(id int) (*Task, error) {
	return s.repo.GetByID(id)
}

func (s *Service) ProcessTask(id int) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !task.CanProcess() {
		return nil, ErrInvalidTransition
	}

	if err := task.TransitionTo(StateProcessing); err != nil {
		return nil, err
	}
	s.repo.Save(task)

	result := analyzeText(task.Input)
	task.Result = &result

	if err := task.TransitionTo(StateCompleted); err != nil {
		return nil, err
	}
	s.repo.Save(task)

	return task, nil
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
