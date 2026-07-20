package model

// Suit is the suit of a playing card.
type Suit string

// The four suits of a standard deck.
const (
	Hearts   Suit = "heart"
	Spades   Suit = "spade"
	Clubs    Suit = "club"
	Diamonds Suit = "diamond"
)

// Value is the face value of a playing card. Numeric values are represented as
// strings so that a card value is a single type across the domain and the API.
type Value string

// The thirteen values of a standard deck. The ace is the 1, so there is no
// separate "1" value.
const (
	Ace   Value = "ace"
	Two   Value = "2"
	Three Value = "3"
	Four  Value = "4"
	Five  Value = "5"
	Six   Value = "6"
	Seven Value = "7"
	Eight Value = "8"
	Nine  Value = "9"
	Ten   Value = "10"
	Jack  Value = "jack"
	Queen Value = "queen"
	King  Value = "king"
)

// AllSuits lists every suit, in deck order.
var AllSuits = []Suit{Hearts, Spades, Clubs, Diamonds}

// AllValues lists every value, in deck order.
var AllValues = []Value{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

// valuePoints scores every value at face value: the ace is 1, the numeric cards
// score their number, and the court cards run on from the ten.
var valuePoints = map[Value]int{
	Ace: 1, Two: 2, Three: 3, Four: 4, Five: 5, Six: 6, Seven: 7,
	Eight: 8, Nine: 9, Ten: 10, Jack: 11, Queen: 12, King: 13,
}

// Card is a single playing card.
type Card struct {
	Suit  Suit
	Value Value
}

// Points is the card's face value as a number. Suit does not enter into it, so
// the four aces are worth the same. A card whose value is not one of the
// thirteen scores 0, which only a hand-built Card can be: NewCards only ever
// produces values from AllValues.
func (c Card) Points() int {
	return valuePoints[c.Value]
}
