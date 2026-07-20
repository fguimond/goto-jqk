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
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
	decks := NewDeckService(memory.NewDeckStore(), gameStore, testLogger())
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
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
	decks := NewDeckService(memory.NewDeckStore(), gameStore, testLogger())
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

func TestPlayerService_Leaders(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
	decks := NewDeckService(memory.NewDeckStore(), gameStore, testLogger())
	ctx := context.Background()

	g, err := games.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}

	// A game with no players ranks empty, which is not an error.
	leaders, err := svc.Leaders(ctx, g.ID)
	if err != nil {
		t.Fatalf("Leaders returned error: %v", err)
	}
	if len(leaders) != 0 {
		t.Errorf("expected no leaders in a game with no players, got %d", len(leaders))
	}

	alice, err := svc.Create(ctx, g.ID, "Alice")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	bob, err := svc.Create(ctx, g.ID, "Bob")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	carol, err := svc.Create(ctx, g.ID, "Carol")
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

	// The deck is unshuffled and suit-major, so the deals come off the top in
	// value order: Alice takes the ace, 2 and 3 of hearts, Bob the 4 through 8.
	if _, err := svc.Deal(ctx, g.ID, alice.ID, 3); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	if _, err := svc.Deal(ctx, g.ID, bob.ID, 5); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	leaders, err = svc.Leaders(ctx, g.ID)
	if err != nil {
		t.Fatalf("Leaders returned error: %v", err)
	}

	// Bob leads on 4+5+6+7+8, Alice follows on 1+2+3, and Carol, dealt nothing,
	// is ranked last rather than left out.
	want := []Leader{
		{Player: bob, Total: 30},
		{Player: alice, Total: 6},
		{Player: carol, Total: 0},
	}
	if len(leaders) != len(want) {
		t.Fatalf("expected %d leaders, got %d", len(want), len(leaders))
	}
	for i, w := range want {
		if leaders[i].Player.ID != w.Player.ID {
			t.Errorf("expected %s in position %d, got %s", w.Player.Name, i, leaders[i].Player.Name)
		}
		if leaders[i].Total != w.Total {
			t.Errorf("expected %s to total %d, got %d", w.Player.Name, w.Total, leaders[i].Total)
		}
	}

	if _, err := svc.Leaders(ctx, uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

// Players on equal totals keep the order they joined the game in, so successive
// calls rank a settled game the same way.
func TestPlayerService_LeadersTiesHoldJoinOrder(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
	decks := NewDeckService(memory.NewDeckStore(), gameStore, testLogger())
	ctx := context.Background()

	g, err := games.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create game returned error: %v", err)
	}
	first, err := svc.Create(ctx, g.ID, "Alice")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	second, err := svc.Create(ctx, g.ID, "Bob")
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

	// Alice takes the ace and 2 of hearts, Bob the 3: both total 3.
	if _, err := svc.Deal(ctx, g.ID, first.ID, 2); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}
	if _, err := svc.Deal(ctx, g.ID, second.ID, 1); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	for range 3 {
		leaders, err := svc.Leaders(ctx, g.ID)
		if err != nil {
			t.Fatalf("Leaders returned error: %v", err)
		}
		if len(leaders) != 2 {
			t.Fatalf("expected 2 leaders, got %d", len(leaders))
		}
		if leaders[0].Total != 3 || leaders[1].Total != 3 {
			t.Fatalf("expected both players to total 3, got %d and %d", leaders[0].Total, leaders[1].Total)
		}
		if leaders[0].Player.ID != first.ID || leaders[1].Player.ID != second.ID {
			t.Errorf("expected tied players in join order, got %s then %s", leaders[0].Player.Name, leaders[1].Player.Name)
		}
	}
}

func TestPlayerService_CreateAndDelete(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
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
	svc := NewPlayerService(memory.NewGameStore(), testLogger())

	if _, err := svc.Create(context.Background(), uuid.New(), "Alice"); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestPlayerService_DeleteUnknownGame(t *testing.T) {
	svc := NewPlayerService(memory.NewGameStore(), testLogger())

	if err := svc.Delete(context.Background(), uuid.New(), uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

// A player may only be removed through the game they belong to, so deleting
// them via another game must not touch them.
func TestPlayerService_DeleteFromWrongGame(t *testing.T) {
	gameStore := memory.NewGameStore()
	games := NewGameService(gameStore, testLogger())
	svc := NewPlayerService(gameStore, testLogger())
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
