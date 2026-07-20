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

func TestGameService_Cards(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewGameService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	ctx := context.Background()

	g, err := svc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// A game with no decks has an empty game deck, not an error.
	cards, err := svc.Cards(ctx, g.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected no cards, got %d", len(cards))
	}

	first, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	second, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{first.ID, second.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	cards, err = svc.Cards(ctx, g.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}

	// The order is the order the decks were added, each in deck order, so the
	// game deck is exactly the two decks concatenated.
	want := append(model.NewCards(), model.NewCards()...)
	if !slices.Equal(cards, want) {
		t.Errorf("expected the game deck to be the two decks concatenated, got %d cards", len(cards))
	}

	if _, err := svc.Cards(ctx, uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestGameService_Shuffle(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewGameService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	ctx := context.Background()

	// Substitute a deterministic permutation for the random one, so the test can
	// assert the exact resulting order.
	svc.shuffle = func(cards []model.Card) {
		slices.Reverse(cards)
	}

	g, err := svc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// A game with an empty game deck shuffles successfully, it is not an error.
	if _, err := svc.Shuffle(ctx, g.ID); err != nil {
		t.Fatalf("Shuffle returned error: %v", err)
	}

	d, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{d.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	shuffled, err := svc.Shuffle(ctx, g.ID)
	if err != nil {
		t.Fatalf("Shuffle returned error: %v", err)
	}
	if len(shuffled.GameDeck) != 52 {
		t.Errorf("expected 52 cards after the shuffle, got %d", len(shuffled.GameDeck))
	}

	want := model.NewCards()
	slices.Reverse(want)
	if !slices.Equal(shuffled.GameDeck, want) {
		t.Errorf("expected the game deck reversed, got a different order")
	}

	// The shuffle must persist: reading the cards back returns the new order.
	cards, err := svc.Cards(ctx, g.ID)
	if err != nil {
		t.Fatalf("Cards returned error: %v", err)
	}
	if !slices.Equal(cards, want) {
		t.Errorf("expected the shuffled order to persist in the store")
	}

	if _, err := svc.Shuffle(ctx, uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
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
