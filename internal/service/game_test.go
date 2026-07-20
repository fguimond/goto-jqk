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

func TestGameService_SuitCounts(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewGameService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	players := NewPlayerService(gameStore)
	ctx := context.Background()

	g, err := svc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// A game with no decks reports every suit, at 0, rather than an empty list.
	counts, err := svc.SuitCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("SuitCounts returned error: %v", err)
	}
	if len(counts) != 4 {
		t.Fatalf("expected 4 suits, got %d", len(counts))
	}
	for _, c := range counts {
		if c.Remaining != 0 {
			t.Errorf("expected 0 %s, got %d", c.Suit, c.Remaining)
		}
	}

	first, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{first.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	counts, err = svc.SuitCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("SuitCounts returned error: %v", err)
	}
	// The suits come back in deck order, each with a full thirteen.
	if !slices.Equal(counts, []SuitCount{
		{Suit: model.Hearts, Remaining: 13},
		{Suit: model.Spades, Remaining: 13},
		{Suit: model.Clubs, Remaining: 13},
		{Suit: model.Diamonds, Remaining: 13},
	}) {
		t.Errorf("expected thirteen of each suit in deck order, got %v", counts)
	}

	// The unshuffled deck is suit-major, hearts first, so dealing five takes five
	// hearts and leaves the other three suits untouched.
	p, err := players.Create(ctx, g.ID, "Ada")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	if _, err := players.Deal(ctx, g.ID, p.ID, 5); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	counts, err = svc.SuitCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("SuitCounts returned error: %v", err)
	}
	if !slices.Equal(counts, []SuitCount{
		{Suit: model.Hearts, Remaining: 8},
		{Suit: model.Spades, Remaining: 13},
		{Suit: model.Clubs, Remaining: 13},
		{Suit: model.Diamonds, Remaining: 13},
	}) {
		t.Errorf("expected the five dealt cards to come off the hearts, got %v", counts)
	}

	// A second deck lifts the counts past thirteen: they are not capped per suit.
	second, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{second.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	counts, err = svc.SuitCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("SuitCounts returned error: %v", err)
	}
	if !slices.Equal(counts, []SuitCount{
		{Suit: model.Hearts, Remaining: 21},
		{Suit: model.Spades, Remaining: 26},
		{Suit: model.Clubs, Remaining: 26},
		{Suit: model.Diamonds, Remaining: 26},
	}) {
		t.Errorf("expected the second deck to lift the counts past thirteen, got %v", counts)
	}

	// Dealing the deck out entirely still reports all four suits, at 0.
	remaining := 21 + 26*3
	if _, err := players.Deal(ctx, g.ID, p.ID, remaining); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	counts, err = svc.SuitCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("SuitCounts returned error: %v", err)
	}
	if len(counts) != 4 {
		t.Fatalf("expected 4 suits once the deck is empty, got %d", len(counts))
	}
	for _, c := range counts {
		if c.Remaining != 0 {
			t.Errorf("expected 0 %s once the deck is empty, got %d", c.Suit, c.Remaining)
		}
	}

	if _, err := svc.SuitCounts(ctx, uuid.New()); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound for an unknown game, got %v", err)
	}
}

func TestGameService_CardCounts(t *testing.T) {
	gameStore := memory.NewGameStore()
	svc := NewGameService(gameStore)
	decks := NewDeckService(memory.NewDeckStore(), gameStore)
	players := NewPlayerService(gameStore)
	ctx := context.Background()

	g, err := svc.Create(ctx, "Poker")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// A game with no decks reports all fifty-two cards, at 0, rather than an empty
	// list.
	counts, err := svc.CardCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("CardCounts returned error: %v", err)
	}
	if len(counts) != 52 {
		t.Fatalf("expected 52 cards, got %d", len(counts))
	}
	for _, c := range counts {
		if c.Remaining != 0 {
			t.Errorf("expected 0 of the %s of %ss, got %d", c.Value, c.Suit, c.Remaining)
		}
	}

	first, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{first.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	counts, err = svc.CardCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("CardCounts returned error: %v", err)
	}
	if len(counts) != 52 {
		t.Fatalf("expected 52 cards, got %d", len(counts))
	}
	// The clubs come first because the suits are alphabetical, not in deck order,
	// and each suit runs from the king down to the ace.
	if !slices.Equal(counts[:13], []CardCount{
		{Suit: model.Clubs, Value: model.King, Remaining: 1},
		{Suit: model.Clubs, Value: model.Queen, Remaining: 1},
		{Suit: model.Clubs, Value: model.Jack, Remaining: 1},
		{Suit: model.Clubs, Value: model.Ten, Remaining: 1},
		{Suit: model.Clubs, Value: model.Nine, Remaining: 1},
		{Suit: model.Clubs, Value: model.Eight, Remaining: 1},
		{Suit: model.Clubs, Value: model.Seven, Remaining: 1},
		{Suit: model.Clubs, Value: model.Six, Remaining: 1},
		{Suit: model.Clubs, Value: model.Five, Remaining: 1},
		{Suit: model.Clubs, Value: model.Four, Remaining: 1},
		{Suit: model.Clubs, Value: model.Three, Remaining: 1},
		{Suit: model.Clubs, Value: model.Two, Remaining: 1},
		{Suit: model.Clubs, Value: model.Ace, Remaining: 1},
	}) {
		t.Errorf("expected the clubs first, king down to ace, got %v", counts[:13])
	}
	// Spades sort last, so the deck ends on the ace of spades.
	if counts[51] != (CardCount{Suit: model.Spades, Value: model.Ace, Remaining: 1}) {
		t.Errorf("expected the ace of spades last, got %v", counts[51])
	}
	for _, c := range counts {
		if c.Remaining != 1 {
			t.Errorf("expected 1 of the %s of %ss, got %d", c.Value, c.Suit, c.Remaining)
		}
	}

	// The unshuffled deck is suit-major, hearts first in AllValues order, so
	// dealing five takes the ace through the five of hearts and nothing else.
	p, err := players.Create(ctx, g.ID, "Ada")
	if err != nil {
		t.Fatalf("Create player returned error: %v", err)
	}
	if _, err := players.Deal(ctx, g.ID, p.ID, 5); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	counts, err = svc.CardCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("CardCounts returned error: %v", err)
	}
	dealt := []model.Value{model.Ace, model.Two, model.Three, model.Four, model.Five}
	for _, c := range counts {
		want := 1
		if c.Suit == model.Hearts && slices.Contains(dealt, c.Value) {
			want = 0
		}
		if c.Remaining != want {
			t.Errorf("expected %d of the %s of %ss, got %d", want, c.Value, c.Suit, c.Remaining)
		}
	}

	// A second deck lifts the counts past one: they are not capped per card.
	second, err := decks.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Create deck returned error: %v", err)
	}
	if _, err := decks.AddDecks(ctx, g.ID, []uuid.UUID{second.ID}); err != nil {
		t.Fatalf("AddDecks returned error: %v", err)
	}

	counts, err = svc.CardCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("CardCounts returned error: %v", err)
	}
	for _, c := range counts {
		want := 2
		if c.Suit == model.Hearts && slices.Contains(dealt, c.Value) {
			want = 1
		}
		if c.Remaining != want {
			t.Errorf("expected %d of the %s of %ss, got %d", want, c.Value, c.Suit, c.Remaining)
		}
	}

	// Dealing the deck out entirely still reports all fifty-two cards, at 0.
	if _, err := players.Deal(ctx, g.ID, p.ID, 52*2-5); err != nil {
		t.Fatalf("Deal returned error: %v", err)
	}

	counts, err = svc.CardCounts(ctx, g.ID)
	if err != nil {
		t.Fatalf("CardCounts returned error: %v", err)
	}
	if len(counts) != 52 {
		t.Fatalf("expected 52 cards once the deck is empty, got %d", len(counts))
	}
	for _, c := range counts {
		if c.Remaining != 0 {
			t.Errorf("expected 0 of the %s of %ss once the deck is empty, got %d", c.Value, c.Suit, c.Remaining)
		}
	}

	if _, err := svc.CardCounts(ctx, uuid.New()); err != store.ErrNotFound {
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
