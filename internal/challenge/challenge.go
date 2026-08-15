package challenge

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Level string

const (
	LevelSimple Level = "simple"
	LevelMedium Level = "medium"
)

var (
	ErrWordNotFound   = errors.New("word not found")
	ErrInvalidAttempt = errors.New("invalid attempt")
)

type Word struct {
	ID        string `json:"id"`
	Level     Level  `json:"level"`
	Prompt    string `json:"prompt"`
	Scrambled string `json:"scrambled"`
	Answer    string `json:"-"`
}

type Attempt struct {
	ID              string `json:"id"`
	WordID          string `json:"wordId"`
	Level           Level  `json:"level"`
	Prompt          string `json:"prompt"`
	Answer          string `json:"answer"`
	Expected        string `json:"expected"`
	Correct         bool   `json:"correct"`
	DurationSeconds int    `json:"durationSeconds"`
	Score           int    `json:"score"`
}

type Stats struct {
	Attempts        int `json:"attempts"`
	Correct         int `json:"correct"`
	Incorrect       int `json:"incorrect"`
	TotalSeconds    int `json:"totalSeconds"`
	TotalScore      int `json:"totalScore"`
	AverageSeconds  int `json:"averageSeconds"`
	AccuracyPercent int `json:"accuracyPercent"`
}

type Mistake struct {
	WordID     string `json:"wordId"`
	Level      Level  `json:"level"`
	Prompt     string `json:"prompt"`
	Scrambled  string `json:"scrambled"`
	Expected   string `json:"expected"`
	LastAnswer string `json:"lastAnswer"`
	Count      int    `json:"count"`
}

type Catalog struct {
	words []Word
}

func NewCatalog() Catalog {
	return Catalog{words: []Word{
		{ID: "simple-apple", Level: LevelSimple, Prompt: "苹果", Scrambled: "paple", Answer: "apple"},
		{ID: "simple-book", Level: LevelSimple, Prompt: "书", Scrambled: "okob", Answer: "book"},
		{ID: "simple-water", Level: LevelSimple, Prompt: "水", Scrambled: "tawer", Answer: "water"},
		{ID: "medium-bridge", Level: LevelMedium, Prompt: "桥梁", Scrambled: "gderib", Answer: "bridge"},
		{ID: "medium-library", Level: LevelMedium, Prompt: "图书馆", Scrambled: "rybrali", Answer: "library"},
		{ID: "medium-journey", Level: LevelMedium, Prompt: "旅程", Scrambled: "yenruoj", Answer: "journey"},
	}}
}

func (c Catalog) Words(level Level) ([]Word, error) {
	if level != LevelSimple && level != LevelMedium {
		return nil, fmt.Errorf("%w: level", ErrInvalidAttempt)
	}
	words := make([]Word, 0, len(c.words))
	for _, word := range c.words {
		if word.Level == level {
			words = append(words, word)
		}
	}
	return words, nil
}

func (c Catalog) Word(id string) (Word, error) {
	for _, word := range c.words {
		if word.ID == id {
			return word, nil
		}
	}
	return Word{}, ErrWordNotFound
}

type ResultStore struct {
	mu       sync.RWMutex
	nextID   int
	attempts []Attempt
}

func NewResultStore() *ResultStore {
	return &ResultStore{}
}

func (s *ResultStore) Add(attempt Attempt) Attempt {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	attempt.ID = fmt.Sprintf("attempt-%03d", s.nextID)
	s.attempts = append(s.attempts, attempt)
	return attempt
}

func (s *ResultStore) All() []Attempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Attempt(nil), s.attempts...)
}

type Service struct {
	catalog Catalog
	results *ResultStore
}

func NewService(catalog Catalog, results *ResultStore) *Service {
	return &Service{catalog: catalog, results: results}
}

func (s *Service) Words(level Level) ([]Word, error) {
	return s.catalog.Words(level)
}

func (s *Service) Submit(wordID, answer string, durationSeconds int) (Attempt, error) {
	word, err := s.catalog.Word(wordID)
	if err != nil {
		return Attempt{}, err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" || durationSeconds < 0 {
		return Attempt{}, ErrInvalidAttempt
	}
	correct := strings.EqualFold(answer, word.Answer)
	score := 0
	if correct {
		score = 10
		if word.Level == LevelMedium {
			score = 20
		}
	}
	return s.results.Add(Attempt{
		WordID:          word.ID,
		Level:           word.Level,
		Prompt:          word.Prompt,
		Answer:          answer,
		Expected:        word.Answer,
		Correct:         correct,
		DurationSeconds: durationSeconds,
		Score:           score,
	}), nil
}

func (s *Service) History() []Attempt {
	return s.results.All()
}

func (s *Service) Stats() Stats {
	attempts := s.results.All()
	stats := Stats{Attempts: len(attempts)}
	for _, attempt := range attempts {
		stats.TotalSeconds += attempt.DurationSeconds
		stats.TotalScore += attempt.Score
		if attempt.Correct {
			stats.Correct++
		}
	}
	stats.Incorrect = stats.Attempts - stats.Correct
	if stats.Attempts > 0 {
		stats.AverageSeconds = stats.TotalSeconds / stats.Attempts
		stats.AccuracyPercent = stats.Correct * 100 / stats.Attempts
	}
	return stats
}

func (s *Service) Mistakes() []Mistake {
	attempts := s.results.All()
	byWord := make(map[string]Mistake)
	for _, attempt := range attempts {
		if attempt.Correct {
			continue
		}
		word, err := s.catalog.Word(attempt.WordID)
		if err != nil {
			continue
		}
		mistake := byWord[word.ID]
		mistake.WordID = word.ID
		mistake.Level = word.Level
		mistake.Prompt = word.Prompt
		mistake.Scrambled = word.Scrambled
		mistake.Expected = word.Answer
		mistake.LastAnswer = attempt.Answer
		mistake.Count++
		byWord[word.ID] = mistake
	}
	mistakes := make([]Mistake, 0, len(byWord))
	for _, mistake := range byWord {
		mistakes = append(mistakes, mistake)
	}
	sort.Slice(mistakes, func(i, j int) bool { return mistakes[i].WordID < mistakes[j].WordID })
	return mistakes
}
