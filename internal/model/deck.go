package model

import "github.com/google/uuid"

// Deck is a deck of playing cards. A deck may exist on its own; GameID holds
// the zero UUID until the deck is assigned to a game.
type Deck struct {
	ID     uuid.UUID
	GameID uuid.UUID
	Cards  []Card
}

// NewCards returns the 52 cards of a standard deck, suit-major and unshuffled.
func NewCards() []Card {
	cards := make([]Card, 0, len(AllSuits)*len(AllValues))
	for _, suit := range AllSuits {
		for _, value := range AllValues {
			cards = append(cards, Card{Suit: suit, Value: value})
		}
	}
	return cards
}
