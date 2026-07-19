package memory

import (
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

// DeckStore is a concurrency-safe in-memory deck store.
type DeckStore struct {
	mu    sync.RWMutex
	decks map[uuid.UUID]*model.Deck
}

// NewDeckStore returns an empty in-memory deck store.
func NewDeckStore() *DeckStore {
	return &DeckStore{
		decks: make(map[uuid.UUID]*model.Deck),
	}
}

// Create stores a new deck, keyed by its ID.
func (s *DeckStore) Create(d *model.Deck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decks[d.ID] = d
	return nil
}

// GetAll returns the decks for the given IDs, in the order requested, or
// ErrNotFound if any ID is unknown.
func (s *DeckStore) GetAll(ids []uuid.UUID) ([]*model.Deck, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	decks := make([]*model.Deck, 0, len(ids))
	for _, id := range ids {
		d, ok := s.decks[id]
		if !ok {
			return nil, store.ErrNotFound
		}
		decks = append(decks, d)
	}
	return decks, nil
}
