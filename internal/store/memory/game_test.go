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
