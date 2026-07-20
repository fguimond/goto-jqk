package service

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/store"
)

// DeckStore is the persistence behavior DeckService depends on, declared here
// at the point of use.
type DeckStore interface {
	Create(d *model.Deck) error
	List() ([]*model.Deck, error)
	GetAll(ids []uuid.UUID) ([]*model.Deck, error)
}

// DeckAttacher attaches decks to an existing game. It is kept separate from
// GameStore so DeckService depends only on the methods it calls.
type DeckAttacher interface {
	AddDeck(gameID uuid.UUID, d *model.Deck) error
	AddDecks(gameID uuid.UUID, decks []*model.Deck) (*model.Game, error)
}

// DeckService implements deck-related business logic.
type DeckService struct {
	// mu guards every read and write of Deck.GameID and Deck.Cards. The service
	// owns those fields: a deck belongs to exactly one game and surrenders its
	// cards to that game, and enforcing those invariants means checking and
	// setting them without another request slipping in between. The stores read
	// the cards but never write either field.
	mu     sync.Mutex
	store  DeckStore
	games  DeckAttacher
	logger *slog.Logger
}

// NewDeckService wires a DeckService to its backing store and to the game store
// it assigns decks through. A nil logger falls back to slog.Default().
func NewDeckService(s DeckStore, g DeckAttacher, logger *slog.Logger) *DeckService {
	if logger == nil {
		logger = slog.Default()
	}
	return &DeckService{
		store:  s,
		games:  g,
		logger: logger.With(slog.String("component", "deck_service")),
	}
}

// Create builds a new 52-card deck with a freshly generated UUID v4 and
// persists it. If gameID is non-nil the deck is assigned to that game and its
// cards move into that game's deck, leaving it empty; store.ErrNotFound is
// returned when no such game exists.
func (s *DeckService) Create(ctx context.Context, gameID *uuid.UUID) (*model.Deck, error) {
	d := &model.Deck{
		ID:    uuid.New(), // uuid.New generates a random (version 4) UUID.
		Cards: model.NewCards(),
	}
	// Captured up front: attaching to a game empties d.Cards.
	cards := len(d.Cards)
	log := opLogger(s.logger, entityDeck, opCreate)

	// Attach before persisting so an unknown game leaves no orphan deck behind.
	if gameID != nil {
		// Held across the attach so no listing catches the cards sitting in
		// both the deck and the game deck.
		s.mu.Lock()
		if err := s.games.AddDeck(*gameID, d); err != nil {
			s.mu.Unlock()
			log.ErrorContext(ctx, "attach new deck to game failed",
				slog.String("deck_id", d.ID.String()),
				slog.String("game_id", gameID.String()),
				slog.Any("error", err),
			)
			return nil, err
		}
		d.GameID = *gameID
		d.Cards = nil
		s.mu.Unlock()
	}

	if err := s.store.Create(d); err != nil {
		log.ErrorContext(ctx, "create deck failed",
			slog.String("deck_id", d.ID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	attrs := []slog.Attr{
		slog.String("deck_id", d.ID.String()),
		slog.Int("cards", cards),
		slog.Bool("attached", gameID != nil),
	}
	if gameID != nil {
		attrs = append(attrs, slog.String("game_id", gameID.String()))
	}
	log.LogAttrs(ctx, slog.LevelInfo, "deck created", attrs...)
	return d, nil
}

// List returns every deck. It holds the lock because listing reads GameID off
// every stored deck, and this service owns that field.
func (s *DeckService) List(_ context.Context) ([]*model.Deck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.List()
}

// AddDecks attaches existing decks to a game, moving their cards into the
// game's deck and leaving them empty, and returns the updated game. It is
// all-or-nothing: store.ErrNotFound is returned if the game or any deck is
// unknown, and store.ErrConflict if any deck already belongs to a game or is
// listed more than once.
func (s *DeckService) AddDecks(ctx context.Context, gameID uuid.UUID, deckIDs []uuid.UUID) (*model.Game, error) {
	// Both sides change, but the game is the resource the request targets and
	// the one a reader is filtering for, so it is the entity here; the decks
	// involved are named in deck_ids.
	log := opLogger(s.logger, entityGame, opUpdate).With(
		slog.String("game_id", gameID.String()),
		slog.Any("deck_ids", deckIDStrings(deckIDs)),
		slog.Int("deck_count", len(deckIDs)),
	)

	// A deck listed twice cannot be attached twice, so reject it up front
	// rather than letting the second attach fail against the first.
	seen := make(map[uuid.UUID]struct{}, len(deckIDs))
	for _, id := range deckIDs {
		if _, dup := seen[id]; dup {
			log.WarnContext(ctx, "add decks to game rejected: deck listed twice",
				slog.String("deck_id", id.String()),
			)
			return nil, store.ErrConflict
		}
		seen[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	decks, err := s.store.GetAll(deckIDs)
	if err != nil {
		log.ErrorContext(ctx, "add decks to game failed: unknown deck", slog.Any("error", err))
		return nil, err
	}

	// A deck belongs to exactly one game. Check them all before appending any,
	// so a rejected patch attaches nothing.
	for _, d := range decks {
		if d.GameID != uuid.Nil {
			log.WarnContext(ctx, "add decks to game rejected: deck already assigned",
				slog.String("deck_id", d.ID.String()),
				slog.String("owning_game_id", d.GameID.String()),
			)
			return nil, store.ErrConflict
		}
	}

	// Counted before the decks give their cards up below.
	moved := 0
	for _, d := range decks {
		moved += len(d.Cards)
	}

	g, err := s.games.AddDecks(gameID, decks)
	if err != nil {
		log.ErrorContext(ctx, "add decks to game failed", slog.Any("error", err))
		return nil, err
	}

	// Stamp and empty only once the append succeeded, so an unknown game leaves
	// no half-assigned decks behind and there is nothing to roll back. The
	// cards now live in the game deck, so the decks give them up.
	for _, d := range decks {
		d.GameID = gameID
		d.Cards = nil
	}

	log.InfoContext(ctx, "decks added to game",
		slog.String("game_name", g.Name),
		slog.Int("cards_moved", moved),
		slog.Int("game_deck_size", len(g.GameDeck)),
	)
	return g, nil
}

// deckIDStrings renders deck IDs for logging: slog would otherwise emit
// uuid.UUID as a byte array.
func deckIDStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
