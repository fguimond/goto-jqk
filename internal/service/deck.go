package service

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

// DeckStore is the persistence behavior DeckService depends on, declared here
// at the point of use.
type DeckStore interface {
	Create(d *model.Deck) error
	GetAll(ids []uuid.UUID) ([]*model.Deck, error)
}

// DeckAttacher attaches decks to an existing game. It is kept separate from
// GameStore so DeckService depends only on the methods it calls.
type DeckAttacher interface {
	AddDeck(gameID uuid.UUID, d *model.Deck) error
	AddDecks(gameID uuid.UUID, decks []*model.Deck) (*model.Game, error)
}

// DeckService implements deck-related business logic.
type DeckService struct {
	// mu guards every read and write of Deck.GameID. The service owns that
	// field: a deck belongs to exactly one game, and enforcing that invariant
	// means checking and setting it without another request slipping in
	// between. The stores never touch it.
	mu    sync.Mutex
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
		s.mu.Lock()
		d.GameID = *gameID
		s.mu.Unlock()
		if err := s.games.AddDeck(*gameID, d); err != nil {
			return nil, err
		}
	}

	if err := s.store.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}

// AddDecks attaches existing decks to a game and returns the updated game. It
// is all-or-nothing: store.ErrNotFound is returned if the game or any deck is
// unknown, and store.ErrConflict if any deck already belongs to a game or is
// listed more than once.
func (s *DeckService) AddDecks(_ context.Context, gameID uuid.UUID, deckIDs []uuid.UUID) (*model.Game, error) {
	// A deck listed twice cannot be attached twice, so reject it up front
	// rather than letting the second attach fail against the first.
	seen := make(map[uuid.UUID]struct{}, len(deckIDs))
	for _, id := range deckIDs {
		if _, dup := seen[id]; dup {
			return nil, store.ErrConflict
		}
		seen[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	decks, err := s.store.GetAll(deckIDs)
	if err != nil {
		return nil, err
	}

	// A deck belongs to exactly one game. Check them all before appending any,
	// so a rejected patch attaches nothing.
	for _, d := range decks {
		if d.GameID != uuid.Nil {
			return nil, store.ErrConflict
		}
	}

	g, err := s.games.AddDecks(gameID, decks)
	if err != nil {
		return nil, err
	}

	// Stamp only once the append succeeded, so an unknown game leaves no
	// half-assigned decks behind and there is nothing to roll back.
	for _, d := range decks {
		d.GameID = gameID
	}
	return g, nil
}
