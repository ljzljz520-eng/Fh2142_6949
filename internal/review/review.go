package review

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrRecordNotFound      = errors.New("review record not found")
	ErrInvalidConfirmation = errors.New("invalid confirmation")
)

type Record struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Confirmations map[string]string `json:"confirmations"`
}

type Repository struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewRepository(fixtures ...Record) *Repository {
	repository := &Repository{records: make(map[string]Record, len(fixtures))}
	for _, fixture := range fixtures {
		repository.records[fixture.ID] = clone(fixture)
	}
	return repository
}

func (r *Repository) Get(id string) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[id]
	if !ok {
		return Record{}, ErrRecordNotFound
	}
	return clone(record), nil
}

// Update atomically merges a single confirmation into the latest persisted
// record. The whole read-modify-write happens under the repository's write
// lock, so concurrent confirmations for the same record never overwrite one
// another: each update is applied to the freshest state instead of a snapshot
// that may already be stale.
func (r *Repository) Update(id, operator, content string) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[id]
	if !ok {
		return Record{}, ErrRecordNotFound
	}
	updated := clone(record)
	updated.Confirmations[operator] = content
	r.records[id] = updated
	return clone(updated), nil
}

type Service struct {
	repository     *Repository
	snapshotLoaded func()
}

func NewService(repository *Repository, snapshotLoaded func()) *Service {
	return &Service{repository: repository, snapshotLoaded: snapshotLoaded}
}

func (s *Service) Record(id string) (Record, error) {
	return s.repository.Get(id)
}

func (s *Service) Confirm(id, operator, content string) (Record, error) {
	operator = strings.TrimSpace(operator)
	content = strings.TrimSpace(content)
	if operator == "" || content == "" {
		return Record{}, ErrInvalidConfirmation
	}
	// Load a snapshot and run the synchronization hook on it, so callers (and
	// tests) can coordinate concurrent confirmations. The snapshot is only used
	// to verify the record exists before blocking; the actual write happens via
	// the atomic Update below, which always merges into the freshest state.
	if _, err := s.repository.Get(id); err != nil {
		return Record{}, err
	}
	if s.snapshotLoaded != nil {
		s.snapshotLoaded()
	}
	return s.repository.Update(id, operator, content)
}

func clone(record Record) Record {
	copyRecord := record
	copyRecord.Confirmations = make(map[string]string, len(record.Confirmations))
	for operator, content := range record.Confirmations {
		copyRecord.Confirmations[operator] = content
	}
	return copyRecord
}
