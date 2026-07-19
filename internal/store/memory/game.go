// Package memory represents an in-memory store for persistent objects
package memory

import (
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

// GameStore is a concurrency-safe in-memory GameStore.
type GameStore struct {
	mu    sync.RWMutex
	games map[uuid.UUID]*model.Game
}

// NewGameStore returns an empty in-memory game store.
func NewGameStore() *GameStore {
	return &GameStore{
		games: make(map[uuid.UUID]*model.Game),
	}
}

// Create stores a new game, keyed by its ID.
func (s *GameStore) Create(g *model.Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[g.ID] = g
	return nil
}

// Delete removes a game by ID, returning ErrNotFound if it is absent.
func (s *GameStore) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.games[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.games, id)
	return nil
}
