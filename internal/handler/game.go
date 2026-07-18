// Package handler contains the HTTP layer, wiring huma operations to the
// service layer and exposing the OpenAPI schema.
package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/service"
	"github.com/fguimond/goto-jqk/internal/store"
)

// GameHandler exposes the game HTTP endpoints.
type GameHandler struct {
	svc *service.GameService
}

// NewGameHandler creates a GameHandler backed by the given service.
func NewGameHandler(svc *service.GameService) *GameHandler {
	return &GameHandler{svc: svc}
}

// Game is the API representation of a game resource.
type Game struct {
	ID   string `json:"id" format:"uuid" doc:"Unique identifier of the game"`
	Name string `json:"name" doc:"Name of the game"`
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

// DeleteGameInput is the path input for deleting a game.
type DeleteGameInput struct {
	ID string `path:"id" format:"uuid" doc:"Unique identifier of the game"`
}

// DeleteGameOutput carries no body; it results in a 204 response.
type DeleteGameOutput struct{}

// Register attaches the game operations to the API.
func (h *GameHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-game",
		Method:        http.MethodPost,
		Path:          "/api/v1/game",
		Summary:       "Create a game",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusCreated,
	}, h.Create)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-game",
		Method:        http.MethodDelete,
		Path:          "/api/v1/game/{id}",
		Summary:       "Delete a game",
		Tags:          []string{"game"},
		DefaultStatus: http.StatusNoContent,
	}, h.Delete)
}

// Create handles POST /api/v1/game.
func (h *GameHandler) Create(ctx context.Context, in *CreateGameInput) (*CreateGameOutput, error) {
	g, err := h.svc.Create(ctx, in.Body.Name)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create game", err)
	}
	return &CreateGameOutput{
		Body: Game{ID: g.ID.String(), Name: g.Name},
	}, nil
}

// Delete handles DELETE /api/v1/game/{id}.
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
