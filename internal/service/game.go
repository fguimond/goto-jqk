// Package service holds the application's business logic, sitting between the
// HTTP handlers and the storage layer.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

// GameService implements game-related business logic.
type GameService struct {
	store store.GameStore
}

// NewGameService wires a GameService to its backing store.
func NewGameService(s store.GameStore) *GameService {
	return &GameService{store: s}
}

// Create builds a new game with a freshly generated UUID v4 and persists it.
func (s *GameService) Create(_ context.Context, name string) (*model.Game, error) {
	g := &model.Game{
		ID:   uuid.New(), // uuid.New generates a random (version 4) UUID.
		Name: name,
	}
	if err := s.store.Create(g); err != nil {
		return nil, err
	}
	return g, nil
}

// Delete removes a game by its ID.
func (s *GameService) Delete(_ context.Context, id uuid.UUID) error {
	return s.store.Delete(id)
}
