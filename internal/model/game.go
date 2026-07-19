// Package model contains the core domain types.
package model

import "github.com/google/uuid"

// Game is the core domain entity. A game carries a name and the decks of cards
// that have been assigned to it.
type Game struct {
	ID    uuid.UUID
	Name  string
	Decks []*Deck
}
