// Package model contains the core domain types.
package model

import "github.com/google/uuid"

// Game is the core domain entity. A game carries a name, the decks of cards
// that have been assigned to it, the game deck those decks were shuffled into,
// and the players taking part.
//
// Assigning a deck moves its cards: they leave Deck.Cards and are appended to
// GameDeck, so a card is only ever in one place. Decks therefore records which
// decks a game was built from, not where its cards live.
type Game struct {
	ID       uuid.UUID
	Name     string
	Decks    []*Deck
	GameDeck []Card
	Players  []*Player
}
