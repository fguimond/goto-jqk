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

// AddDeck appends a deck to the game with the given ID and its cards to the
// game deck, returning ErrNotFound if the game is absent. The lookup and the
// append happen under a single lock so callers never mutate a stored game
// outside the store's synchronization.
//
// The deck's cards are read, not cleared: emptying the deck is the deck
// service's, since it owns every write to a deck.
func (s *GameStore) AddDeck(gameID uuid.UUID, d *model.Deck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return store.ErrNotFound
	}
	g.Decks = append(g.Decks, d)
	g.GameDeck = append(g.GameDeck, d.Cards...)
	return nil
}

// AddDecks appends decks to the game with the given ID and their cards to the
// game deck, returning a snapshot of the updated game, or ErrNotFound if the
// game is absent. Callers are responsible for validating that the decks may be
// attached; the store only records the association and takes the cards.
//
// As in AddDeck, the decks' cards are read, not cleared. The snapshot is a
// copy, so callers never hold the stored game.
func (s *GameStore) AddDecks(gameID uuid.UUID, decks []*model.Deck) (*model.Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return nil, store.ErrNotFound
	}
	g.Decks = append(g.Decks, decks...)
	for _, d := range decks {
		g.GameDeck = append(g.GameDeck, d.Cards...)
	}

	return snapshotGame(g), nil
}

// Shuffle permutes the game deck of the game with the given ID, returning a
// snapshot of the updated game, or ErrNotFound if the game is absent. The
// lookup and the permutation happen under a single lock so callers never mutate
// a stored game outside the store's synchronization.
//
// The permutation itself is supplied by the caller, keeping the choice of
// randomness out of the store. A game with an empty game deck is a no-op, not
// an error. The snapshot is a copy, so callers never hold the stored game.
func (s *GameStore) Shuffle(id uuid.UUID, shuffle func([]model.Card)) (*model.Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	shuffle(g.GameDeck)

	return snapshotGame(g), nil
}

// AddPlayer appends a player to the game with the given ID and returns a
// snapshot of the updated game, or ErrNotFound if the game is absent. The
// lookup and the append happen under a single lock so callers never mutate a
// stored game outside the store's synchronization.
func (s *GameStore) AddPlayer(gameID uuid.UUID, p *model.Player) (*model.Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return nil, store.ErrNotFound
	}
	g.Players = append(g.Players, p)

	return snapshotGame(g), nil
}

// RemovePlayer drops a player from the game with the given ID, returning
// ErrNotFound if either the game or the player is absent.
func (s *GameStore) RemovePlayer(gameID, playerID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return store.ErrNotFound
	}
	i := slices.IndexFunc(g.Players, func(p *model.Player) bool {
		return p.ID == playerID
	})
	if i < 0 {
		return store.ErrNotFound
	}
	g.Players = slices.Delete(g.Players, i, i+1)
	return nil
}

// DealCards moves count cards off the top of the game deck into the hand of the
// player with the given ID, returning the cards dealt. ErrNotFound is returned
// if either the game or the player is absent, and ErrConflict if the game deck
// holds fewer than count cards, in which case nothing is dealt.
//
// The lookup, the check and the transfer happen under a single lock, so both
// sides of the move are consistent and callers never mutate a stored game
// outside the store's synchronization.
//
// The dealt cards are cloned rather than handed out as a window onto the game
// deck's backing array. Reslicing forward as this does abandons the head, which
// nothing then writes to, so the clone is not strictly required today — it holds
// the invariant unconditionally instead of resting on that argument, which a
// switch to a compacting delete would quietly invalidate.
func (s *GameStore) DealCards(gameID, playerID uuid.UUID, count int) ([]model.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[gameID]
	if !ok {
		return nil, store.ErrNotFound
	}
	i := slices.IndexFunc(g.Players, func(p *model.Player) bool {
		return p.ID == playerID
	})
	if i < 0 {
		return nil, store.ErrNotFound
	}
	if count > len(g.GameDeck) {
		return nil, store.ErrConflict
	}

	dealt := slices.Clone(g.GameDeck[:count])
	g.GameDeck = g.GameDeck[count:]
	g.Players[i].Cards = append(g.Players[i].Cards, dealt...)

	return dealt, nil
}

// snapshotGame copies a stored game and its slices, so callers never hold the
// stored game or the backing arrays the store keeps appending to. Callers must
// hold the lock.
func snapshotGame(g *model.Game) *model.Game {
	snapshot := *g
	snapshot.Decks = slices.Clone(g.Decks)
	snapshot.GameDeck = slices.Clone(g.GameDeck)
	snapshot.Players = slices.Clone(g.Players)
	return &snapshot
}

// Get returns a snapshot of the game with the given ID, or ErrNotFound if it is
// absent. The snapshot is a copy, so callers never hold the stored game.
func (s *GameStore) Get(id uuid.UUID) (*model.Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.games[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return snapshotGame(g), nil
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
		games = append(games, snapshotGame(g))
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
