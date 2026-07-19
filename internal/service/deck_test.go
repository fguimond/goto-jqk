package service

import (
	"context"
	"sync"
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

func TestDeckService_AddDecks(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewDeckService(memory.NewDeckStore(), gameStore)
	gameSvc := NewGameService(gameStore)
	ctx := context.Background()

	g, err := gameSvc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	first, err := svc.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	second, err := svc.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}

	updated, err := svc.AddDecks(ctx, g.ID, []uuid.UUID{first.ID, second.ID})
	if err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}
	if len(updated.Decks) != 2 {
		t.Fatalf("expected 2 decks on the game, got %d", len(updated.Decks))
	}

	// An unknown deck fails the whole patch, so the spare stays unassigned.
	spare, err := svc.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := svc.AddDecks(ctx, g.ID, []uuid.UUID{spare.ID, uuid.New()}); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown deck, got %v", err)
	}
	if spare.GameID != uuid.Nil {
		t.Errorf("expected the spare deck to stay unassigned, got %v", spare.GameID)
	}

	if _, err := svc.AddDecks(ctx, uuid.New(), []uuid.UUID{spare.ID}); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}

	if _, err := svc.AddDecks(ctx, g.ID, []uuid.UUID{spare.ID, spare.ID}); err != store.ErrConflict {
		t.Errorf("expected ErrConflict for a duplicated deck, got %v", err)
	}

	if _, err := svc.AddDecks(ctx, g.ID, []uuid.UUID{first.ID}); err != store.ErrConflict {
		t.Errorf("expected ErrConflict for an already assigned deck, got %v", err)
	}
}

func TestDeckService_List(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewDeckService(memory.NewDeckStore(), gameStore)
	gameSvc := NewGameService(gameStore)
	ctx := context.Background()

	decks, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(decks) != 0 {
		t.Fatalf("expected no decks, got %d", len(decks))
	}

	g, err := gameSvc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	if _, err := svc.Create(ctx, nil); err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	assigned, err := svc.Create(ctx, &g.ID)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}

	decks, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("expected 2 decks, got %d", len(decks))
	}

	// The listing must carry each deck's assignment and its full card count.
	for _, d := range decks {
		if len(d.Cards) != 52 {
			t.Errorf("expected 52 cards on deck %v, got %d", d.ID, len(d.Cards))
		}
		want := uuid.Nil
		if d.ID == assigned.ID {
			want = g.ID
		}
		if d.GameID != want {
			t.Errorf("expected deck %v to have game id %v, got %v", d.ID, want, d.GameID)
		}
	}
}

// TestDeckService_AddDecksConcurrent races several games for the same deck.
// Exactly one may win: the service checks and sets Deck.GameID under its own
// lock precisely so this cannot interleave. Run with -race.
func TestDeckService_AddDecksConcurrent(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewDeckService(memory.NewDeckStore(), gameStore)
	gameSvc := NewGameService(gameStore)
	ctx := context.Background()

	d, err := svc.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}

	const contenders = 8
	games := make([]uuid.UUID, contenders)
	for i := range games {
		g, err := gameSvc.Create(ctx, "Poker")
		if err != nil {
			t.Fatalf("Create game returned error: %v", err)
		}
		games[i] = g.ID
	}

	var wg sync.WaitGroup
	errs := make([]error, contenders)
	for i, gameID := range games {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = svc.AddDecks(ctx, gameID, []uuid.UUID{d.ID})
		}()
	}
	wg.Wait()

	won := 0
	for _, err := range errs {
		switch err {
		case nil:
			won++
		case store.ErrConflict:
		default:
			t.Errorf("expected nil or ErrConflict, got %v", err)
		}
	}
	if won != 1 {
		t.Errorf("expected exactly 1 game to win the deck, got %d", won)
	}
}
