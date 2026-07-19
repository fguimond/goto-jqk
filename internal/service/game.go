// Package service holds the application's business logic, sitting between the
// HTTP handlers and the storage layer.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// GameStore is the persistence behavior GameService depends on. It is declared
// here, at the point of use, so the service owns the abstraction it consumes
// rather than importing one defined alongside a concrete implementation.
type GameStore interface {
	Create(g *model.Game) error
	Get(id uuid.UUID) (*model.Game, error)
	List() ([]*model.Game, error)
	Delete(id uuid.UUID) error
}

// GameService implements game-related business logic.
type GameService struct {
	store GameStore
}

// NewGameService wires a GameService to its backing store.
func NewGameService(s GameStore) *GameService {
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

// List returns every game.
func (s *GameService) List(_ context.Context) ([]*model.Game, error) {
	return s.store.List()
}

// Cards returns the cards in the game's deck, in the order they sit in it,
// which is the order they were added and so the order they will be dealt.
// store.ErrNotFound is returned when no such game exists.
func (s *GameService) Cards(_ context.Context, id uuid.UUID) ([]model.Card, error) {
	g, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	return g.GameDeck, nil
}

// Delete removes a game by its ID.
func (s *GameService) Delete(_ context.Context, id uuid.UUID) error {
	return s.store.Delete(id)
}
