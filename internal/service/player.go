package service

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// PlayerStore is the persistence behavior PlayerService depends on, declared
// here at the point of use. Players have no standalone lifecycle — they are
// only ever reached through the game they belong to — so they are stored as
// part of the game aggregate rather than in a store of their own.
type PlayerStore interface {
	AddPlayer(gameID uuid.UUID, p *model.Player) (*model.Game, error)
	RemovePlayer(gameID, playerID uuid.UUID) error
	DealCards(gameID, playerID uuid.UUID, count int) ([]model.Card, error)
	PlayerCards(gameID, playerID uuid.UUID) ([]model.Card, error)
	Players(gameID uuid.UUID) ([]*model.Player, error)
}

// Leader is a player's standing in a game: the player together with the total
// face value of the cards they hold.
type Leader struct {
	Player *model.Player
	Total  int
}

// PlayerService implements player-related business logic.
type PlayerService struct {
	games  PlayerStore
	logger *slog.Logger
}

// NewPlayerService wires a PlayerService to the game store it records players
// against. A nil logger falls back to slog.Default().
func NewPlayerService(g PlayerStore, logger *slog.Logger) *PlayerService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PlayerService{
		games:  g,
		logger: logger.With(slog.String("component", "player_service")),
	}
}

// Create builds a new player with a freshly generated UUID v4 and adds them to
// the given game, returning store.ErrNotFound when no such game exists. A new
// player starts with no cards.
func (s *PlayerService) Create(ctx context.Context, gameID uuid.UUID, name string) (*model.Player, error) {
	p := &model.Player{
		ID:     uuid.New(), // uuid.New generates a random (version 4) UUID.
		GameID: gameID,
		Name:   name,
	}
	log := opLogger(s.logger, entityPlayer, opCreate)
	g, err := s.games.AddPlayer(gameID, p)
	if err != nil {
		log.ErrorContext(ctx, "create player failed",
			slog.String("game_id", gameID.String()),
			slog.String("player_id", p.ID.String()),
			slog.String("player_name", name),
			slog.Any("error", err),
		)
		return nil, err
	}
	log.InfoContext(ctx, "player created",
		slog.String("game_id", gameID.String()),
		slog.String("player_id", p.ID.String()),
		slog.String("player_name", p.Name),
		slog.Int("players_in_game", len(g.Players)),
	)
	return p, nil
}

// Deal moves count cards off the top of the game's deck into the player's hand
// and returns the cards dealt. It is all-or-nothing: store.ErrConflict is
// returned if the game deck holds fewer than count cards, and nothing is dealt.
// store.ErrNotFound is returned if either the game or the player is unknown.
//
// The move happens entirely inside the store, which holds both the game deck
// and the player's hand, so the two sides never disagree.
func (s *PlayerService) Deal(ctx context.Context, gameID, playerID uuid.UUID, count int) ([]model.Card, error) {
	log := opLogger(s.logger, entityPlayer, opUpdate)
	cards, err := s.games.DealCards(gameID, playerID, count)
	if err != nil {
		log.ErrorContext(ctx, "deal cards failed",
			slog.String("game_id", gameID.String()),
			slog.String("player_id", playerID.String()),
			slog.Int("requested", count),
			slog.Any("error", err),
		)
		return nil, err
	}
	log.InfoContext(ctx, "cards dealt",
		slog.String("game_id", gameID.String()),
		slog.String("player_id", playerID.String()),
		slog.Int("requested", count),
		slog.Int("dealt", len(cards)),
	)
	return cards, nil
}

// Cards returns the player's whole hand, in the order the cards were dealt,
// rather than the cards any one deal produced. A player who has been dealt
// nothing holds an empty hand, which is not an error. store.ErrNotFound is
// returned if either the game or the player is unknown.
func (s *PlayerService) Cards(_ context.Context, gameID, playerID uuid.UUID) ([]model.Card, error) {
	return s.games.PlayerCards(gameID, playerID)
}

// Leaders returns the game's players ranked by the total face value of the
// cards they hold, highest total first. Cards score at face value only, so the
// suit never matters. store.ErrNotFound is returned if the game is unknown.
//
// Every player is ranked, including one who has been dealt nothing: they place
// last with a total of 0 rather than being left out. A game with no players
// ranks empty, which is not an error.
//
// The sort is stable, so players on equal totals keep the order they joined in
// and successive calls rank a settled game the same way.
func (s *PlayerService) Leaders(_ context.Context, gameID uuid.UUID) ([]Leader, error) {
	players, err := s.games.Players(gameID)
	if err != nil {
		return nil, err
	}

	leaders := make([]Leader, 0, len(players))
	for _, p := range players {
		total := 0
		for _, c := range p.Cards {
			total += c.Points()
		}
		leaders = append(leaders, Leader{Player: p, Total: total})
	}
	slices.SortStableFunc(leaders, func(a, b Leader) int {
		return cmp.Compare(b.Total, a.Total)
	})
	return leaders, nil
}

// Delete removes a player from a game, returning store.ErrNotFound if either
// the game or the player is unknown.
func (s *PlayerService) Delete(ctx context.Context, gameID, playerID uuid.UUID) error {
	log := opLogger(s.logger, entityPlayer, opDelete)
	if err := s.games.RemovePlayer(gameID, playerID); err != nil {
		log.ErrorContext(ctx, "delete player failed",
			slog.String("game_id", gameID.String()),
			slog.String("player_id", playerID.String()),
			slog.Any("error", err),
		)
		return err
	}
	log.InfoContext(ctx, "player deleted",
		slog.String("game_id", gameID.String()),
		slog.String("player_id", playerID.String()),
	)
	return nil
}
