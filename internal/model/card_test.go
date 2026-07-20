package model

import "testing"

func TestCardPoints(t *testing.T) {
	want := map[Value]int{
		Ace: 1, Two: 2, Three: 3, Four: 4, Five: 5, Six: 6, Seven: 7,
		Eight: 8, Nine: 9, Ten: 10, Jack: 11, Queen: 12, King: 13,
	}

	// Every value in deck order, so a value gained or lost here fails as loudly
	// as one scored wrongly.
	if len(AllValues) != len(want) {
		t.Fatalf("expected %d values to score, got %d", len(want), len(AllValues))
	}
	for _, v := range AllValues {
		got := Card{Suit: Hearts, Value: v}.Points()
		if got != want[v] {
			t.Errorf("expected %q to score %d, got %d", v, want[v], got)
		}
	}

	// Suit does not enter into the score, so a value is worth the same in all four.
	for _, s := range AllSuits {
		if got := (Card{Suit: s, Value: King}).Points(); got != 13 {
			t.Errorf("expected the king of %q to score 13, got %d", s, got)
		}
	}
}

// A full deck is the sum of every value once per suit: 91 a suit, 364 in all.
func TestCardPointsWholeDeck(t *testing.T) {
	total := 0
	for _, c := range NewCards() {
		total += c.Points()
	}
	if total != 364 {
		t.Errorf("expected a deck to total 364, got %d", total)
	}
}
