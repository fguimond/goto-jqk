package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// DeckStore is the persistence behavior DeckService depends on, declared here
// at the point of use.
type DeckStore interface {
	Create(d *model.Deck) error
}

// DeckAttacher attaches a deck to an existing game. It is kept separate from
// GameStore so DeckService depends only on the one method it calls.
type DeckAttacher interface {
	AddDeck(gameID uuid.UUID, d *model.Deck) error
}

// DeckService implements deck-related business logic.
type DeckService struct {
	store DeckStore
	games DeckAttacher
}

// NewDeckService wires a DeckService to its backing store and to the game store
// it assigns decks through.
func NewDeckService(s DeckStore, g DeckAttacher) *DeckService {
	return &DeckService{store: s, games: g}
}

// Create builds a new 52-card deck with a freshly generated UUID v4 and
// persists it. If gameID is non-nil the deck is assigned to that game, and
// store.ErrNotFound is returned when no such game exists.
func (s *DeckService) Create(_ context.Context, gameID *uuid.UUID) (*model.Deck, error) {
	d := &model.Deck{
		ID:    uuid.New(), // uuid.New generates a random (version 4) UUID.
		Cards: model.NewCards(),
	}

	// Attach before persisting so an unknown game leaves no orphan deck behind.
	if gameID != nil {
		d.GameID = *gameID
		if err := s.games.AddDeck(*gameID, d); err != nil {
			return nil, err
		}
	}

	if err := s.store.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}
