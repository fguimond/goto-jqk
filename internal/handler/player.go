package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
	"github.com/fguimond/goto-jqk/internal/service"
	"github.com/fguimond/goto-jqk/internal/store"
)

// PlayerHandler exposes the player HTTP endpoints. Players are game-scoped, so
// every route hangs off /api/v1/games/{gameId}.
type PlayerHandler struct {
	svc *service.PlayerService
}

// NewPlayerHandler creates a PlayerHandler backed by the given service.
func NewPlayerHandler(svc *service.PlayerService) *PlayerHandler {
	return &PlayerHandler{svc: svc}
}

// Card is the API representation of a playing card.
type Card struct {
	Suit  string `json:"suit" doc:"Suit of the card" example:"heart"`
	Value string `json:"value" doc:"Face value of the card" example:"ace"`
}

// Player is the API representation of a player resource.
type Player struct {
	ID     string `json:"id" format:"uuid" doc:"Unique identifier of the player"`
	GameID string `json:"gameId" format:"uuid" doc:"Game the player belongs to"`
	Name   string `json:"name" doc:"Name of the player"`
	Cards  []Card `json:"cards" doc:"Cards held by the player"`
}

// newPlayer builds the API representation of a domain player.
func newPlayer(p *model.Player) Player {
	// Built empty rather than nil so a player with no cards serializes the
	// field as [], not null.
	cards := make([]Card, 0, len(p.Cards))
	for _, c := range p.Cards {
		cards = append(cards, Card{Suit: string(c.Suit), Value: string(c.Value)})
	}
	return Player{
		ID:     p.ID.String(),
		GameID: p.GameID.String(),
		Name:   p.Name,
		Cards:  cards,
	}
}

// CreatePlayerInput is the path and request body for creating a player.
type CreatePlayerInput struct {
	GameID string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
	Body   struct {
		Name string `json:"name" minLength:"1" maxLength:"255" doc:"Name of the player" example:"Alice"`
	}
}

// CreatePlayerOutput is the response for a created player.
type CreatePlayerOutput struct {
	Body Player
}

// DeletePlayerInput is the path input for removing a player from a game.
type DeletePlayerInput struct {
	GameID   string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
	PlayerID string `path:"playerId" format:"uuid" doc:"Unique identifier of the player"`
}

// DeletePlayerOutput carries no body; it results in a 204 response.
type DeletePlayerOutput struct{}

// DealCardsInput is the path and request body for dealing cards to a player.
type DealCardsInput struct {
	GameID   string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
	PlayerID string `path:"playerId" format:"uuid" doc:"Unique identifier of the player"`
	Body     struct {
		// No default: huma marks a non-pointer field required regardless, so a
		// documented default would advertise behavior the API does not honor.
		Count int `json:"count" minimum:"1" doc:"Number of cards to deal" example:"2"`
	}
}

// DealCardsOutput is the response carrying the cards dealt by this call, rather
// than the player's whole hand: those are the cards the operation produced.
type DealCardsOutput struct {
	Body []Card
}

// ListPlayerCardsInput is the path input for listing a player's cards.
type ListPlayerCardsInput struct {
	GameID   string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
	PlayerID string `path:"playerId" format:"uuid" doc:"Unique identifier of the player"`
}

// ListPlayerCardsOutput is the response carrying the player's whole hand, as
// against DealCardsOutput, which carries only the cards one deal produced.
type ListPlayerCardsOutput struct {
	Body []Card
}

// Leader is the API representation of a player's standing in a game. The hand
// behind the total is deliberately left out: it is served by the player's cards
// endpoint, which keeps a standing a fixed size per player.
type Leader struct {
	PlayerID string `json:"playerId" format:"uuid" doc:"Unique identifier of the player"`
	Name     string `json:"name" doc:"Name of the player"`
	Total    int    `json:"total" doc:"Total face value of the cards the player holds" example:"34"`
}

// ListLeadersInput is the path input for ranking a game's players.
type ListLeadersInput struct {
	GameID string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
}

// ListLeadersOutput is the response carrying the ranked players.
type ListLeadersOutput struct {
	Body []Leader
}

// Register attaches the player operations to the API.
func (h *PlayerHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-player",
		Method:        http.MethodPost,
		Path:          "/api/v1/games/{gameId}/players",
		Summary:       "Create a player",
		Description:   "Creates a player and adds them to the game. A new player holds no cards.",
		Tags:          []string{"player"},
		DefaultStatus: http.StatusCreated,
	}, h.Create)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-player",
		Method:        http.MethodDelete,
		Path:          "/api/v1/games/{gameId}/players/{playerId}",
		Summary:       "Remove a player from a game",
		Tags:          []string{"player"},
		DefaultStatus: http.StatusNoContent,
	}, h.Delete)

	huma.Register(api, huma.Operation{
		OperationID:   "deal-cards",
		Method:        http.MethodPost,
		Path:          "/api/v1/games/{gameId}/players/{playerId}/cards",
		Summary:       "Deal cards to a player",
		Description:   "Deals cards off the top of the game deck into the player's hand and returns the cards dealt. The deal is all-or-nothing: if the game deck holds fewer cards than requested, none are dealt. Successive deals continue down the deck rather than repeating cards.",
		Tags:          []string{"player"},
		DefaultStatus: http.StatusCreated,
	}, h.Deal)

	huma.Register(api, huma.Operation{
		OperationID:   "list-player-cards",
		Method:        http.MethodGet,
		Path:          "/api/v1/games/{gameId}/players/{playerId}/cards",
		Summary:       "List a player's cards",
		Description:   "Lists the cards in the player's hand, in the order they were dealt. This is the whole hand, accumulated across every deal, rather than the cards a single deal produced. A player who has been dealt nothing holds an empty list.",
		Tags:          []string{"player"},
		DefaultStatus: http.StatusOK,
	}, h.ListCards)

	huma.Register(api, huma.Operation{
		OperationID:   "list-leaders",
		Method:        http.MethodGet,
		Path:          "/api/v1/games/{gameId}/leaders",
		Summary:       "Rank a game's players",
		Description:   "Lists the players in the game with the total value of the cards each one holds, highest total first. Cards count at face value only: the ace is 1, the numeric cards score their number, the jack 11, the queen 12 and the king 13. Suit does not affect the total. Every player is listed, including one who has been dealt nothing, who places last with a total of 0. Players on equal totals keep the order they joined the game in.",
		Tags:          []string{"player"},
		DefaultStatus: http.StatusOK,
	}, h.ListLeaders)
}

// Create handles POST /api/v1/games/{gameId}/players.
func (h *PlayerHandler) Create(ctx context.Context, in *CreatePlayerInput) (*CreatePlayerOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}

	p, err := h.svc.Create(ctx, gameID, in.Body.Name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game not found")
		}
		return nil, huma.Error500InternalServerError("failed to create player", err)
	}

	return &CreatePlayerOutput{Body: newPlayer(p)}, nil
}

// Deal handles POST /api/v1/games/{gameId}/players/{playerId}/cards.
func (h *PlayerHandler) Deal(ctx context.Context, in *DealCardsInput) (*DealCardsOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}
	playerID, err := uuid.Parse(in.PlayerID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid player id", err)
	}

	dealt, err := h.svc.Deal(ctx, gameID, playerID, in.Body.Count)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return nil, huma.Error404NotFound("game or player not found")
		case errors.Is(err, store.ErrConflict):
			return nil, huma.Error409Conflict("not enough cards in the game deck")
		}
		return nil, huma.Error500InternalServerError("failed to deal cards", err)
	}

	// Built empty rather than nil so a deal of no cards serializes as [], not null.
	body := make([]Card, 0, len(dealt))
	for _, c := range dealt {
		body = append(body, Card{Suit: string(c.Suit), Value: string(c.Value)})
	}
	return &DealCardsOutput{Body: body}, nil
}

// ListCards handles GET /api/v1/games/{gameId}/players/{playerId}/cards.
func (h *PlayerHandler) ListCards(ctx context.Context, in *ListPlayerCardsInput) (*ListPlayerCardsOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}
	playerID, err := uuid.Parse(in.PlayerID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid player id", err)
	}

	cards, err := h.svc.Cards(ctx, gameID, playerID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game or player not found")
		}
		return nil, huma.Error500InternalServerError("failed to list cards", err)
	}

	// Built empty rather than nil so an empty hand serializes as [], not null.
	body := make([]Card, 0, len(cards))
	for _, c := range cards {
		body = append(body, Card{Suit: string(c.Suit), Value: string(c.Value)})
	}
	return &ListPlayerCardsOutput{Body: body}, nil
}

// ListLeaders handles GET /api/v1/games/{gameId}/leaders.
func (h *PlayerHandler) ListLeaders(ctx context.Context, in *ListLeadersInput) (*ListLeadersOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}

	leaders, err := h.svc.Leaders(ctx, gameID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game not found")
		}
		return nil, huma.Error500InternalServerError("failed to rank players", err)
	}

	// Built empty rather than nil so a game with no players serializes as [], not null.
	body := make([]Leader, 0, len(leaders))
	for _, l := range leaders {
		body = append(body, Leader{
			PlayerID: l.Player.ID.String(),
			Name:     l.Player.Name,
			Total:    l.Total,
		})
	}
	return &ListLeadersOutput{Body: body}, nil
}

// Delete handles DELETE /api/v1/games/{gameId}/players/{playerId}.
func (h *PlayerHandler) Delete(ctx context.Context, in *DeletePlayerInput) (*DeletePlayerOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}
	playerID, err := uuid.Parse(in.PlayerID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid player id", err)
	}

	if err := h.svc.Delete(ctx, gameID, playerID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game or player not found")
		}
		return nil, huma.Error500InternalServerError("failed to remove player", err)
	}
	return &DeletePlayerOutput{}, nil
}
