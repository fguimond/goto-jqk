package memory

import (
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
