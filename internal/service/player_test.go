package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/store"
	"github.com/fguimond/goto-jqk/internal/store/memory"
)

func TestPlayerService_CreateAndDelete(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore)
	svc := NewPlayerService(gameStore)
	ctx := context.Background()

	g, err := games.Create(ctx, "Chess")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}

	p, err := svc.Create(ctx, g.ID, "Alice")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if p.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", p.Name)
	}
	if p.ID.Version() != 4 {
		t.Errorf("expected a UUID v4, got version %d", p.ID.Version())
	}
	if p.GameID != g.ID {
		t.Errorf("expected game ID %v, got %v", g.ID, p.GameID)
	}
	if len(p.Cards) != 0 {
		t.Errorf("expected a new player to hold no cards, got %d", len(p.Cards))
	}

	// The player should show up on the game they were added to.
	listed, err := games.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed) != 1 || len(listed[0].Players) != 1 || listed[0].Players[0].ID != p.ID {
		t.Fatalf("expected the game to carry player %v", p.ID)
	}

	if err := svc.Delete(ctx, g.ID, p.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if err := svc.Delete(ctx, g.ID, p.ID); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound on second delete, got %v", err)
	}

	listed, err = games.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed[0].Players) != 0 {
		t.Errorf("expected no players after the delete, got %d", len(listed[0].Players))
	}
}

func TestPlayerService_CreateUnknownGame(t *testing.T) {
	svc := NewPlayerService(memory.NewGameStore())

	if _, err := svc.Create(context.Background(), uuid.New(), "Alice"); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestPlayerService_DeleteUnknownGame(t *testing.T) {
	svc := NewPlayerService(memory.NewGameStore())

	if err := svc.Delete(context.Background(), uuid.New(), uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

// A player may only be removed through the game they belong to, so deleting
// them via another game must not touch them.
func TestPlayerService_DeleteFromWrongGame(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore)
	svc := NewPlayerService(gameStore)
	ctx := context.Background()

	first, err := games.Create(ctx, "Chess")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	second, err := games.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}

	p, err := svc.Create(ctx, first.ID, "Alice")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.Delete(ctx, second.ID, p.ID); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound deleting from the wrong game, got %v", err)
	}
	if err := svc.Delete(ctx, first.ID, p.ID); err != nil {
		t.Errorf("expected the player to still exist on their own game, got %v", err)
	}
}
