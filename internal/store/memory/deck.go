package memory

import (
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
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
