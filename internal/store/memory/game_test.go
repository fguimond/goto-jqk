package memory

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

func TestGameStore_AddDeck(t *testing.T) {
	s := NewGameStore()
	g := &model.Game{ID: uuid.New(), Name: "Poker"}
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	d := &model.Deck{ID: uuid.New(), GameID: g.ID, Cards: model.NewCards()}
	if err := s.AddDeck(g.ID, d); err != nil {
		t.Fatalf("AddDeck returned error: %v", err)
	}

	stored := s.games[g.ID]
	if len(stored.Decks) != 1 {
		t.Fatalf("expected 1 deck on the game, got %d", len(stored.Decks))
	}
	if stored.Decks[0].ID != d.ID {
		t.Errorf("expected deck %v, got %v", d.ID, stored.Decks[0].ID)
	}
	if len(stored.GameDeck) != 52 {
		t.Errorf("expected 52 cards in the game deck, got %d", len(stored.GameDeck))
	}

	if err := s.AddDeck(uuid.New(), d); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestGameStore_Get(t *testing.T) {
	s := NewGameStore()
	g := &model.Game{ID: uuid.New(), Name: "Poker", GameDeck: model.NewCards()}
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got, err := s.Get(g.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != g.ID || len(got.GameDeck) != 52 {
		t.Errorf("expected game %v with 52 cards, got %v with %d", g.ID, got.ID, len(got.GameDeck))
	}

	// The snapshot must be a copy: mutating it leaves the store untouched.
	got.GameDeck = got.GameDeck[:0]
	if len(s.games[g.ID].GameDeck) != 52 {
		t.Errorf("expected the stored game deck to still have 52 cards, got %d", len(s.games[g.ID].GameDeck))
	}

	if _, err := s.Get(uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestGameStore_Shuffle(t *testing.T) {
	s := NewGameStore()
	g := &model.Game{ID: uuid.New(), Name: "Poker", GameDeck: model.NewCards()}
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// A deterministic permutation, so the test asserts the store applied exactly
	// what it was handed rather than merely that something changed.
	got, err := s.Shuffle(g.ID, func(cards []model.Card) {
		slices.Reverse(cards)
	})
	if err != nil {
		t.Fatalf("Shuffle returned error: %v", err)
	}

	want := model.NewCards()
	slices.Reverse(want)
	if !slices.Equal(s.games[g.ID].GameDeck, want) {
		t.Errorf("expected the stored game deck reversed, got a different order")
	}
	if !slices.Equal(got.GameDeck, want) {
		t.Errorf("expected the returned game deck reversed, got a different order")
	}

	// The snapshot must be a copy: mutating it leaves the store untouched.
	got.GameDeck = got.GameDeck[:0]
	if len(s.games[g.ID].GameDeck) != 52 {
		t.Errorf("expected the stored game deck to still have 52 cards, got %d", len(s.games[g.ID].GameDeck))
	}

	// An empty game deck is a no-op, not an error.
	empty := &model.Game{ID: uuid.New(), Name: "Chess"}
	if err := s.Create(empty); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := s.Shuffle(empty.ID, func(cards []model.Card) { slices.Reverse(cards) }); err != nil {
		t.Errorf("expected shuffling an empty game deck to succeed, got %v", err)
	}

	if _, err := s.Shuffle(uuid.New(), func([]model.Card) {}); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestGameStore_DealCards(t *testing.T) {
	s := NewGameStore()
	p := &model.Player{ID: uuid.New(), Name: "Alice"}
	g := &model.Game{
		ID:       uuid.New(),
		Name:     "Poker",
		GameDeck: model.NewCards(),
		Players:  []*model.Player{p},
	}
	g.Players[0].GameID = g.ID
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	want := model.NewCards()
	dealt, err := s.DealCards(g.ID, p.ID, 5)
	if err != nil {
		t.Fatalf("DealCards returned error: %v", err)
	}
	if !slices.Equal(dealt, want[:5]) {
		t.Errorf("expected the first 5 cards of the game deck, got a different set")
	}

	// Both sides of the move must agree: the deck gave up exactly what the hand
	// received.
	stored := s.games[g.ID]
	if len(stored.GameDeck) != 47 {
		t.Errorf("expected 47 cards left in the game deck, got %d", len(stored.GameDeck))
	}
	if !slices.Equal(stored.GameDeck, want[5:]) {
		t.Errorf("expected the game deck to resume at the 6th card")
	}
	if !slices.Equal(stored.Players[0].Cards, want[:5]) {
		t.Errorf("expected the player to hold the 5 dealt cards")
	}

	// Mutating the returned cards must leave the store untouched. This holds
	// today even without the clone in DealCards, because reslicing forward
	// abandons the head of the array rather than writing over it — so this is a
	// regression guard on the invariant, not a check that the clone is present.
	for i := range dealt {
		dealt[i] = model.Card{}
	}
	if !slices.Equal(s.games[g.ID].Players[0].Cards, want[:5]) {
		t.Errorf("expected the stored hand to survive mutation of the returned cards")
	}
	if !slices.Equal(s.games[g.ID].GameDeck, want[5:]) {
		t.Errorf("expected the stored game deck to survive mutation of the returned cards")
	}

	// Dealing more than the deck holds deals nothing at all.
	if _, err := s.DealCards(g.ID, p.ID, 48); err != store.ErrConflict {
		t.Errorf("expected ErrConflict when the deck is too short, got %v", err)
	}
	if len(s.games[g.ID].GameDeck) != 47 {
		t.Errorf("expected the rejected deal to leave 47 cards, got %d", len(s.games[g.ID].GameDeck))
	}

	if _, err := s.DealCards(uuid.New(), p.ID, 1); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
	if _, err := s.DealCards(g.ID, uuid.New(), 1); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown player, got %v", err)
	}
}

func TestGameStore_List(t *testing.T) {
	s := NewGameStore()
	g := &model.Game{ID: uuid.New(), Name: "Poker", Decks: []*model.Deck{{ID: uuid.New()}}}
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	games, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}

	// The listing must hand back copies: mutating one leaves the store untouched.
	games[0].Name = "Chess"
	games[0].Decks = append(games[0].Decks, &model.Deck{ID: uuid.New()})
	stored := s.games[g.ID]
	if stored.Name != "Poker" {
		t.Errorf("expected the stored game to still be named Poker, got %q", stored.Name)
	}
	if len(stored.Decks) != 1 {
		t.Errorf("expected the stored game to still have 1 deck, got %d", len(stored.Decks))
	}
}

func TestGameStore_AddDecks(t *testing.T) {
	s := NewGameStore()
	g := &model.Game{ID: uuid.New(), Name: "Poker"}
	if err := s.Create(g); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	first := &model.Deck{ID: uuid.New(), Cards: model.NewCards()}
	second := &model.Deck{ID: uuid.New(), Cards: model.NewCards()}
	snapshot, err := s.AddDecks(g.ID, []*model.Deck{first, second})
	if err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}
	if len(snapshot.Decks) != 2 {
		t.Fatalf("expected 2 decks on the snapshot, got %d", len(snapshot.Decks))
	}

	// Both decks' cards land in the game deck, in the order the decks came in.
	if len(snapshot.GameDeck) != 104 {
		t.Fatalf("expected 104 cards in the game deck, got %d", len(snapshot.GameDeck))
	}

	// The store records the association but never writes to a deck; assigning
	// GameID and emptying Cards are the deck service's to own.
	if first.GameID != uuid.Nil || second.GameID != uuid.Nil {
		t.Errorf("expected the store to leave GameID alone, got %v and %v", first.GameID, second.GameID)
	}
	if len(first.Cards) != 52 || len(second.Cards) != 52 {
		t.Errorf("expected the store to leave Cards alone, got %d and %d", len(first.Cards), len(second.Cards))
	}

	// The snapshot must be a copy: mutating it leaves the store untouched.
	snapshot.Decks = append(snapshot.Decks, &model.Deck{ID: uuid.New()})
	snapshot.GameDeck = append(snapshot.GameDeck, model.Card{Suit: model.Hearts, Value: model.Ace})
	if len(s.games[g.ID].Decks) != 2 {
		t.Errorf("expected the stored game to still have 2 decks, got %d", len(s.games[g.ID].Decks))
	}
	if len(s.games[g.ID].GameDeck) != 104 {
		t.Errorf("expected the stored game deck to still have 104 cards, got %d", len(s.games[g.ID].GameDeck))
	}

	if _, err := s.AddDecks(uuid.New(), []*model.Deck{{ID: uuid.New()}}); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}
