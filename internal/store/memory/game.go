// Package memory represents an in-memory store for persistent objects
package memory

import (
	"bytes"
	"slices"
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

// AddDeck appends a deck to the game with the given ID, returning ErrNotFound
// if the game is absent. The lookup and the append happen under a single lock
// so callers never mutate a stored game outside the store's synchronization.
func (s *GameStore) AddDeck(gameID uuid.UUID, d *model.Deck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return store.ErrNotFound
	}
	g.Decks = append(g.Decks, d)
	return nil
}

// AddDecks appends decks to the game with the given ID and returns a snapshot
// of the updated game, or ErrNotFound if the game is absent. Callers are
// responsible for validating that the decks may be attached; the store only
// records the association.
//
// The snapshot is a copy, so callers never hold the stored game.
func (s *GameStore) AddDecks(gameID uuid.UUID, decks []*model.Deck) (*model.Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return nil, store.ErrNotFound
	}
	g.Decks = append(g.Decks, decks...)

	snapshot := *g
	snapshot.Decks = slices.Clone(g.Decks)
	return &snapshot, nil
}

// List returns every stored game, ordered by ID. Map iteration order is
// randomized, so the sort is what keeps successive listings stable.
//
// Each game is a snapshot, so callers never hold a stored game.
func (s *GameStore) List() ([]*model.Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	games := make([]*model.Game, 0, len(s.games))
	for _, g := range s.games {
		snapshot := *g
		snapshot.Decks = slices.Clone(g.Decks)
		games = append(games, &snapshot)
	}
	slices.SortFunc(games, func(a, b *model.Game) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return games, nil
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
