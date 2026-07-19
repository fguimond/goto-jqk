package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/store"
	"github.com/fguimond/goto-jqk/internal/store/memory"
)

func TestDeckService_Create(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewDeckService(memory.NewDeckStore(), gameStore)
	gameSvc := NewGameService(gameStore)
	ctx := context.Background()

	d, err := svc.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(d.Cards) != 52 {
		t.Errorf("expected 52 cards, got %d", len(d.Cards))
	}
	if d.ID.Version() != 4 {
		t.Errorf("expected a UUID v4, got version %d", d.ID.Version())
	}
	if d.GameID != uuid.Nil {
		t.Errorf("expected an unassigned deck, got game id %v", d.GameID)
	}

	g, err := gameSvc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	assigned, err := svc.Create(ctx, &g.ID)
	if err != nil {
		t.Fatalf("Create with game returned error: %v", err)
	}
	if assigned.GameID != g.ID {
		t.Errorf("expected game id %v, got %v", g.ID, assigned.GameID)
	}

	unknown := uuid.New()
	if _, err := svc.Create(ctx, &unknown); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}
