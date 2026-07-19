package service

import (
	"context"
	"testing"

	"github.com/fguimond/goto-jqk/internal/store"
	"github.com/fguimond/goto-jqk/internal/store/memory"
)

func TestGameService_CreateAndDelete(t *testing.T) {
	svc := NewGameService(memory.NewGameStore())
	ctx := context.Background()

	g, err := svc.Create(ctx, "Chess")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if g.Name != "Chess" {
		t.Errorf("expected name %q, got %q", "Chess", g.Name)
	}
	if g.ID.Version() != 4 {
		t.Errorf("expected a UUID v4, got version %d", g.ID.Version())
	}

	if err := svc.Delete(ctx, g.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if err := svc.Delete(ctx, g.ID); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestGameService_List(t *testing.T) {
	svc := NewGameService(memory.NewGameStore())
	ctx := context.Background()

	games, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(games) != 0 {
		t.Fatalf("expected no games, got %d", len(games))
	}

	first, err := svc.Create(ctx, "Chess")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := svc.Create(ctx, "Poker"); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	games, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}

	// The order is by ID, not by insertion, so look the game up by name.
	var found bool
	for _, g := range games {
		if g.ID == first.ID && g.Name == "Chess" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the created game %v in the listing", first.ID)
	}

	if err := svc.Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	games, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(games) != 1 {
		t.Errorf("expected 1 game after the delete, got %d", len(games))
	}
}
