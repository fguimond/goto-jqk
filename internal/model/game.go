// Package model contains the core domain types.
package model

import "github.com/google/uuid"

// Game is the core domain entity. A game carries a name, the decks of cards
// that have been assigned to it, and the players taking part.
type Game struct {
	ID      uuid.UUID
	Name    string
	Decks   []*Deck
	Players []*Player
}
