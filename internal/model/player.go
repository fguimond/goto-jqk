package model

import "github.com/google/uuid"

// Player is a participant in a game, holding the cards dealt to them. Unlike a
// Deck, a player never exists on their own: they are created on a game-scoped
// route, so GameID is always set.
type Player struct {
	ID     uuid.UUID
	GameID uuid.UUID
	Name   string
	Cards  []Card
}
