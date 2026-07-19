// Package handler contains the HTTP layer, wiring huma operations to the
// service layer and exposing the OpenAPI schema.
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

// GameHandler exposes the game HTTP endpoints. Assigning decks is a game-scoped
// route but deck-assignment logic lives on DeckService, so the handler holds
// both services.
type GameHandler struct {
	svc   *service.GameService
	decks *service.DeckService
}

// NewGameHandler creates a GameHandler backed by the given services.
func NewGameHandler(svc *service.GameService, decks *service.DeckService) *GameHandler {
	return &GameHandler{svc: svc, decks: decks}
}

// Game is the API representation of a game resource. The game deck is reported
// as a count so that listing games stays a fixed size per game; the cards
// themselves are served by the game's cards endpoint.
type Game struct {
	ID                string   `json:"id" format:"uuid" doc:"Unique identifier of the game"`
	Name              string   `json:"name" doc:"Name of the game"`
	Decks             []string `json:"decks" doc:"IDs of the decks assigned to the game"`
	GameDeckRemaining int      `json:"gameDeckRemaining" doc:"Number of cards left in the game deck"`
	Players           []string `json:"players" doc:"IDs of the players in the game"`
}

// newGame builds the API representation of a domain game. The ID slices are
// built empty rather than nil so a game with none serializes them as [], not
// null.
func newGame(g *model.Game) Game {
	decks := make([]string, 0, len(g.Decks))
	for _, d := range g.Decks {
		decks = append(decks, d.ID.String())
	}
	players := make([]string, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p.ID.String())
	}
	return Game{
		ID:                g.ID.String(),
		Name:              g.Name,
		Decks:             decks,
		GameDeckRemaining: len(g.GameDeck),
		Players:           players,
	}
}

// CreateGameInput is the request body for creating a game.
type CreateGameInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"255" doc:"Name of the game" example:"Chess"`
	}
}

// CreateGameOutput is the response for a created game.
type CreateGameOutput struct {
	Body Game
}

// ListGamesInput carries no parameters; listing takes no arguments.
type ListGamesInput struct{}

// ListGamesOutput is the response carrying every game.
type ListGamesOutput struct {
	Body []Game
}

// DeleteGameInput is the path input for deleting a game.
type DeleteGameInput struct {
	ID string `path:"id" format:"uuid" doc:"Unique identifier of the game"`
}

// DeleteGameOutput carries no body; it results in a 204 response.
type DeleteGameOutput struct{}

// DeckPatchOp is a single RFC 6902 operation against a game's deck list. Only
// "add" against the append pointer "/-" is supported; the enum tags land in the
// OpenAPI schema, so anything else is rejected during request validation.
type DeckPatchOp struct {
	Op    string `json:"op" enum:"add" doc:"Patch operation. Only \"add\" is supported." example:"add"`
	Path  string `json:"path" enum:"/-" doc:"JSON Pointer. Only the append pointer \"/-\" is supported." example:"/-"`
	Value string `json:"value" format:"uuid" doc:"ID of an existing, unassigned deck" example:"1b4e28ba-2fa1-11d2-883f-0016d3cca427"`
}

// AddGameDecksInput is the path and patch document for adding decks to a game.
type AddGameDecksInput struct {
	GameID string        `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
	Body   []DeckPatchOp `minItems:"1" doc:"RFC 6902 patch document"`
}

// AddGameDecksOutput is the response carrying the updated game.
type AddGameDecksOutput struct {
	Body Game
}

// ListGameCardsInput is the path input for listing a game's cards.
type ListGameCardsInput struct {
	GameID string `path:"gameId" format:"uuid" doc:"Unique identifier of the game"`
}

// ListGameCardsOutput is the response carrying the game deck's cards.
type ListGameCardsOutput struct {
	Body []Card
}

// Register attaches the game operations to the API.
func (h *GameHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-game",
		Method:        http.MethodPost,
		Path:          "/api/v1/games",
		Summary:       "Create a game",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusCreated,
	}, h.Create)

	huma.Register(api, huma.Operation{
		OperationID:   "list-games",
		Method:        http.MethodGet,
		Path:          "/api/v1/games",
		Summary:       "List games",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusOK,
	}, h.List)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-game",
		Method:        http.MethodDelete,
		Path:          "/api/v1/games/{id}",
		Summary:       "Delete a game",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusNoContent,
	}, h.Delete)

	huma.Register(api, huma.Operation{
		OperationID:   "list-game-cards",
		Method:        http.MethodGet,
		Path:          "/api/v1/games/{gameId}/cards",
		Summary:       "List a game's cards",
		Description:   "Lists the cards in the game deck, in the order they sit in it: decks are appended in the order they were added, each contributing its cards in deck order. This is the order the cards will be dealt in.",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusOK,
	}, h.ListCards)

	huma.Register(api, huma.Operation{
		OperationID:   "add-game-decks",
		Method:        http.MethodPatch,
		Path:          "/api/v1/games/{gameId}/decks",
		Summary:       "Add decks to a game",
		Description:   "Applies an RFC 6902 patch document to the game's deck list. Only the \"add\" operation against the append pointer \"/-\" is supported. Each added deck surrenders its cards to the game deck, leaving the deck empty. The patch is applied atomically: if any deck is unknown or already assigned to a game, none are added.",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusOK,
	}, h.AddDecks)
}

// Create handles POST /api/v1/games.
func (h *GameHandler) Create(ctx context.Context, in *CreateGameInput) (*CreateGameOutput, error) {
	g, err := h.svc.Create(ctx, in.Body.Name)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create game", err)
	}
	return &CreateGameOutput{Body: newGame(g)}, nil
}

// List handles GET /api/v1/games.
func (h *GameHandler) List(ctx context.Context, _ *ListGamesInput) (*ListGamesOutput, error) {
	games, err := h.svc.List(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list games", err)
	}
	// Built empty rather than nil so an empty listing serializes as [], not null.
	body := make([]Game, 0, len(games))
	for _, g := range games {
		body = append(body, newGame(g))
	}
	return &ListGamesOutput{Body: body}, nil
}

// ListCards handles GET /api/v1/games/{gameId}/cards.
func (h *GameHandler) ListCards(ctx context.Context, in *ListGameCardsInput) (*ListGameCardsOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}

	cards, err := h.svc.Cards(ctx, gameID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game not found")
		}
		return nil, huma.Error500InternalServerError("failed to list cards", err)
	}

	// Built empty rather than nil so an empty game deck serializes as [], not null.
	body := make([]Card, 0, len(cards))
	for _, c := range cards {
		body = append(body, Card{Suit: string(c.Suit), Value: string(c.Value)})
	}
	return &ListGameCardsOutput{Body: body}, nil
}

// Delete handles DELETE /api/v1/games/{id}.
func (h *GameHandler) Delete(ctx context.Context, in *DeleteGameInput) (*DeleteGameOutput, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game not found")
		}
		return nil, huma.Error500InternalServerError("failed to delete game", err)
	}
	return &DeleteGameOutput{}, nil
}

// AddDecks handles PATCH /api/v1/games/{gameId}/decks.
func (h *GameHandler) AddDecks(ctx context.Context, in *AddGameDecksInput) (*AddGameDecksOutput, error) {
	gameID, err := uuid.Parse(in.GameID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid game id", err)
	}

	deckIDs := make([]uuid.UUID, 0, len(in.Body))
	for _, op := range in.Body {
		id, err := uuid.Parse(op.Value)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid deck id", err)
		}
		deckIDs = append(deckIDs, id)
	}

	g, err := h.decks.AddDecks(ctx, gameID, deckIDs)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return nil, huma.Error404NotFound("game or deck not found")
		case errors.Is(err, store.ErrConflict):
			return nil, huma.Error409Conflict("deck is already assigned to a game")
		}
		return nil, huma.Error500InternalServerError("failed to add decks", err)
	}

	return &AddGameDecksOutput{Body: newGame(g)}, nil
}
