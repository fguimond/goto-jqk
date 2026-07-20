package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// PlayerStore is the persistence behavior PlayerService depends on, declared
// here at the point of use. Players have no standalone lifecycle — they are
// only ever reached through the game they belong to — so they are stored as
// part of the game aggregate rather than in a store of their own.
type PlayerStore interface {
	AddPlayer(gameID uuid.UUID, p *model.Player) (*model.Game, error)
	RemovePlayer(gameID, playerID uuid.UUID) error
	DealCards(gameID, playerID uuid.UUID, count int) ([]model.Card, error)
}

// PlayerService implements player-related business logic.
type PlayerService struct {
	games PlayerStore
}

// NewPlayerService wires a PlayerService to the game store it records players
// against.
func NewPlayerService(g PlayerStore) *PlayerService {
	return &PlayerService{games: g}
}

// Create builds a new player with a freshly generated UUID v4 and adds them to
// the given game, returning store.ErrNotFound when no such game exists. A new
// player starts with no cards.
func (s *PlayerService) Create(_ context.Context, gameID uuid.UUID, name string) (*model.Player, error) {
	p := &model.Player{
		ID:     uuid.New(), // uuid.New generates a random (version 4) UUID.
		GameID: gameID,
		Name:   name,
	}
	if _, err := s.games.AddPlayer(gameID, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Deal moves count cards off the top of the game's deck into the player's hand
// and returns the cards dealt. It is all-or-nothing: store.ErrConflict is
// returned if the game deck holds fewer than count cards, and nothing is dealt.
// store.ErrNotFound is returned if either the game or the player is unknown.
//
// The move happens entirely inside the store, which holds both the game deck
// and the player's hand, so the two sides never disagree.
func (s *PlayerService) Deal(_ context.Context, gameID, playerID uuid.UUID, count int) ([]model.Card, error) {
	return s.games.DealCards(gameID, playerID, count)
}

// Delete removes a player from a game, returning store.ErrNotFound if either
// the game or the player is unknown.
func (s *PlayerService) Delete(_ context.Context, gameID, playerID uuid.UUID) error {
	return s.games.RemovePlayer(gameID, playerID)
}
