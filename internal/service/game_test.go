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
