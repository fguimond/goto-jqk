package model

import "testing"

func TestNewCards(t *testing.T) {
	cards := NewCards()

	if len(cards) != 52 {
		t.Fatalf("expected 52 cards, got %d", len(cards))
	}

	// Card is comparable, so a set of every card catches duplicates as well as
	// a suit or value appearing more or fewer times than it should.
	seen := make(map[Card]struct{}, len(cards))
	for _, c := range cards {
		seen[c] = struct{}{}
	}
	if len(seen) != 52 {
		t.Errorf("expected 52 distinct cards, got %d", len(seen))
	}
}
