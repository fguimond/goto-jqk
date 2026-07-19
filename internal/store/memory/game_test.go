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

	d := &model.Deck{ID: uuid.New(), GameID: g.ID}
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

	first := &model.Deck{ID: uuid.New()}
	second := &model.Deck{ID: uuid.New()}
	snapshot, err := s.AddDecks(g.ID, []*model.Deck{first, second})
	if err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}
	if len(snapshot.Decks) != 2 {
		t.Fatalf("expected 2 decks on the snapshot, got %d", len(snapshot.Decks))
	}

	// The store records the association but never assigns GameID; that is the
	// deck service's to own.
	if first.GameID != uuid.Nil || second.GameID != uuid.Nil {
		t.Errorf("expected the store to leave GameID alone, got %v and %v", first.GameID, second.GameID)
	}

	// The snapshot must be a copy: mutating it leaves the store untouched.
	snapshot.Decks = append(snapshot.Decks, &model.Deck{ID: uuid.New()})
	if len(s.games[g.ID].Decks) != 2 {
		t.Errorf("expected the stored game to still have 2 decks, got %d", len(s.games[g.ID].Decks))
	}

	if _, err := s.AddDecks(uuid.New(), []*model.Deck{{ID: uuid.New()}}); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}
