// Package service holds the application's business logic, sitting between the
// HTTP handlers and the storage layer.
package service

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// GameStore is the persistence behavior GameService depends on. It is declared
// here, at the point of use, so the service owns the abstraction it consumes
// rather than importing one defined alongside a concrete implementation.
type GameStore interface {
	Create(g *model.Game) error
	Get(id uuid.UUID) (*model.Game, error)
	List() ([]*model.Game, error)
	Delete(id uuid.UUID) error
	Shuffle(id uuid.UUID, shuffle func([]model.Card)) (*model.Game, error)
}

// GameService implements game-related business logic.
//
// The service owns the shuffle: the store synchronizes the mutation but takes
// the permutation as an argument, so the choice of randomness lives here. Tests
// in this package substitute a deterministic permutation.
type GameService struct {
	store   GameStore
	shuffle func([]model.Card)
	logger  *slog.Logger
}

// NewGameService wires a GameService to its backing store. A nil logger falls
// back to slog.Default().
func NewGameService(s GameStore, logger *slog.Logger) *GameService {
	if logger == nil {
		logger = slog.Default()
	}
	return &GameService{
		store:   s,
		shuffle: shuffleCards,
		logger:  logger.With(slog.String("component", "game_service")),
	}
}

// shuffleCards permutes cards in place. The math/rand/v2 global source is
// seeded randomly at startup, so successive runs deal differently without any
// seeding of our own.
func shuffleCards(cards []model.Card) {
	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
}

// Create builds a new game with a freshly generated UUID v4 and persists it.
func (s *GameService) Create(ctx context.Context, name string) (*model.Game, error) {
	g := &model.Game{
		ID:   uuid.New(), // uuid.New generates a random (version 4) UUID.
		Name: name,
	}
	log := opLogger(s.logger, entityGame, opCreate)
	if err := s.store.Create(g); err != nil {
		log.ErrorContext(ctx, "create game failed",
			slog.String("game_id", g.ID.String()),
			slog.String("game_name", name),
			slog.Any("error", err),
		)
		return nil, err
	}
	log.InfoContext(ctx, "game created",
		slog.String("game_id", g.ID.String()),
		slog.String("game_name", g.Name),
	)
	return g, nil
}

// List returns every game.
func (s *GameService) List(_ context.Context) ([]*model.Game, error) {
	return s.store.List()
}

// Cards returns the cards in the game's deck, in the order they sit in it, which
// is the order they will be dealt. That starts out as the order they were added
// and holds until the deck is shuffled. store.ErrNotFound is returned when no
// such game exists.
func (s *GameService) Cards(_ context.Context, id uuid.UUID) ([]model.Card, error) {
	g, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	return g.GameDeck, nil
}

// SuitCount is the number of undealt cards of one suit left in a game deck.
type SuitCount struct {
	Suit      model.Suit
	Remaining int
}

// SuitCounts reports how many cards of each suit are left undealt in the game's
// deck. Every suit is listed, in deck order, including one with nothing left,
// which reports 0. Cards already dealt to players are not counted, and a game
// holding several decks can leave more than thirteen of a suit, so the counts
// are not capped. store.ErrNotFound is returned when no such game exists.
func (s *GameService) SuitCounts(ctx context.Context, id uuid.UUID) ([]SuitCount, error) {
	cards, err := s.Cards(ctx, id)
	if err != nil {
		return nil, err
	}

	tally := make(map[model.Suit]int, len(model.AllSuits))
	for _, c := range cards {
		tally[c.Suit]++
	}

	// Ranged over AllSuits rather than the tally so the order is deck order and
	// a suit that has run out still reports, at 0.
	counts := make([]SuitCount, 0, len(model.AllSuits))
	for _, suit := range model.AllSuits {
		counts = append(counts, SuitCount{Suit: suit, Remaining: tally[suit]})
	}
	return counts, nil
}

// CardCount is the number of undealt copies of one card left in a game deck.
type CardCount struct {
	Suit      model.Suit
	Value     model.Value
	Remaining int
}

// CardCounts reports how many copies of each card are left undealt in the game's
// deck. All fifty-two cards are listed, ordered by suit alphabetically and then
// by face value from the king down to the ace, including a card with nothing
// left, which reports 0. Cards already dealt to players are not counted, and a
// game holding several decks can leave more than one copy of a card, so the
// counts are not capped. store.ErrNotFound is returned when no such game exists.
func (s *GameService) CardCounts(ctx context.Context, id uuid.UUID) ([]CardCount, error) {
	cards, err := s.Cards(ctx, id)
	if err != nil {
		return nil, err
	}

	// Card is comparable, so the tally keys on the whole card rather than nesting
	// a map per suit.
	tally := make(map[model.Card]int, len(model.AllSuits)*len(model.AllValues))
	for _, c := range cards {
		tally[c]++
	}

	// A sorted copy: AllSuits is in deck order, which NewCards and SuitCounts rely
	// on, so it must not be sorted in place.
	suits := slices.Sorted(slices.Values(model.AllSuits))

	// Ranged over the suits and values rather than the tally so the order is fixed
	// and a card that has run out still reports, at 0. AllValues runs ace to king,
	// which is ascending face value, so it is walked backwards to descend.
	counts := make([]CardCount, 0, len(suits)*len(model.AllValues))
	for _, suit := range suits {
		for i := len(model.AllValues) - 1; i >= 0; i-- {
			value := model.AllValues[i]
			counts = append(counts, CardCount{
				Suit:      suit,
				Value:     value,
				Remaining: tally[model.Card{Suit: suit, Value: value}],
			})
		}
	}
	return counts, nil
}

// Shuffle randomizes the order of the game's deck and returns the updated game.
// store.ErrNotFound is returned when no such game exists.
func (s *GameService) Shuffle(ctx context.Context, id uuid.UUID) (*model.Game, error) {
	log := opLogger(s.logger, entityGame, opUpdate)
	g, err := s.store.Shuffle(id, s.shuffle)
	if err != nil {
		log.ErrorContext(ctx, "shuffle game deck failed",
			slog.String("game_id", id.String()),
			slog.Any("error", err),
		)
		return nil, err
	}
	log.InfoContext(ctx, "game deck shuffled",
		slog.String("game_id", g.ID.String()),
		slog.String("game_name", g.Name),
		slog.Int("cards", len(g.GameDeck)),
	)
	return g, nil
}

// Delete removes a game by its ID.
func (s *GameService) Delete(ctx context.Context, id uuid.UUID) error {
	log := opLogger(s.logger, entityGame, opDelete)
	if err := s.store.Delete(id); err != nil {
		log.ErrorContext(ctx, "delete game failed",
			slog.String("game_id", id.String()),
			slog.Any("error", err),
		)
		return err
	}
	log.InfoContext(ctx, "game deleted", slog.String("game_id", id.String()))
	return nil
}
