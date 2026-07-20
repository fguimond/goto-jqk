package service

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
	"github.com/fguimond/goto-jqk/internal/store/memory"
)

func TestPlayerService_Deal(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore)
	svc := NewPlayerService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	ctx := context.Background()

	g, err := games.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	p, err := svc.Create(ctx, g.ID, "Alice")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	d, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{d.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	// The deal comes off the top of the game deck, in order.
	want := model.NewCards()
	dealt, err := svc.Deal(ctx, g.ID, p.ID, 5)
	if err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	if !slices.Equal(dealt, want[:5]) {
		t.Errorf("expected the first 5 cards of the game deck, got a different set")
	}

	cards, err := games.Cards(ctx, g.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if len(cards) != 47 {
		t.Errorf("expected 47 cards left in the game deck, got %d", len(cards))
	}

	// A second deal continues down the deck rather than repeating cards.
	dealt, err = svc.Deal(ctx, g.ID, p.ID, 3)
	if err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	if !slices.Equal(dealt, want[5:8]) {
		t.Errorf("expected the next 3 cards of the game deck, got a different set")
	}

	// Both deals landed in the player's hand.
	listed, err := games.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed[0].Players[0].Cards) != 8 {
		t.Errorf("expected the player to hold 8 cards, got %d", len(listed[0].Players[0].Cards))
	}

	// Dealing more than the deck holds deals nothing at all.
	if _, err := svc.Deal(ctx, g.ID, p.ID, 99); err != store.ErrConflict {
		t.Errorf("expected ErrConflict when the deck is too short, got %v", err)
	}
	cards, err = games.Cards(ctx, g.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if len(cards) != 44 {
		t.Errorf("expected the rejected deal to leave 44 cards, got %d", len(cards))
	}

	if _, err := svc.Deal(ctx, uuid.New(), p.ID, 1); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
	if _, err := svc.Deal(ctx, g.ID, uuid.New(), 1); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown player, got %v", err)
	}
}

func TestPlayerService_Cards(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore)
	svc := NewPlayerService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	ctx := context.Background()

	g, err := games.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	p, err := svc.Create(ctx, g.ID, "Alice")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	d, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{d.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	// A player who has been dealt nothing holds an empty hand, not an error.
	hand, err := svc.Cards(ctx, g.ID, p.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if len(hand) != 0 {
		t.Errorf("expected a new player to hold no cards, got %d", len(hand))
	}

	want := model.NewCards()
	if _, err := svc.Deal(ctx, g.ID, p.ID, 5); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	hand, err = svc.Cards(ctx, g.ID, p.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if !slices.Equal(hand, want[:5]) {
		t.Errorf("expected the 5 dealt cards, got a different set")
	}

	// The hand accumulates across deals rather than reporting the last one.
	if _, err := svc.Deal(ctx, g.ID, p.ID, 3); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	hand, err = svc.Cards(ctx, g.ID, p.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if !slices.Equal(hand, want[:8]) {
		t.Errorf("expected all 8 dealt cards in deal order, got a different set")
	}

	if _, err := svc.Cards(ctx, uuid.New(), p.ID); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
	if _, err := svc.Cards(ctx, g.ID, uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown player, got %v", err)
	}
}

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
