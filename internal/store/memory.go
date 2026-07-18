package store

import (
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// MemoryGameStore is a concurrency-safe in-memory GameStore.
type MemoryGameStore struct {
	mu    sync.RWMutex
	games map[uuid.UUID]*model.Game
}

// NewMemoryGameStore returns an empty in-memory game store.
func NewMemoryGameStore() *MemoryGameStore {
	return &MemoryGameStore{
		games: make(map[uuid.UUID]*model.Game),
	}
}

// Create stores a new game, keyed by its ID.
func (s *MemoryGameStore) Create(g *model.Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[g.ID] = g
	return nil
}

// Delete removes a game by ID, returning ErrNotFound if it is absent.
func (s *MemoryGameStore) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.games[id]; !ok {
		return ErrNotFound
	}
	delete(s.games, id)
	return nil
}
