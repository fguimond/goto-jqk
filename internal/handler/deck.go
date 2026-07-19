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

// DeckHandler exposes the deck HTTP endpoints.
type DeckHandler struct {
	svc *service.DeckService
}

// NewDeckHandler creates a DeckHandler backed by the given service.
func NewDeckHandler(svc *service.DeckService) *DeckHandler {
	return &DeckHandler{svc: svc}
}

// Deck is the API representation of a deck resource.
type Deck struct {
	ID        string `json:"id" format:"uuid" doc:"Unique identifier of the deck"`
	GameID    string `json:"gameId,omitempty" format:"uuid" doc:"Game the deck is assigned to, if any"`
	Remaining int    `json:"remaining" doc:"Number of cards left in the deck"`
}

// newDeck builds the API representation of a domain deck. An unassigned deck
// carries the zero UUID, which the omitempty tag drops from the response.
func newDeck(d *model.Deck) Deck {
	out := Deck{ID: d.ID.String(), Remaining: len(d.Cards)}
	if d.GameID != uuid.Nil {
		out.GameID = d.GameID.String()
	}
	return out
}

// CreateDeckInput is the request body for creating a deck.
type CreateDeckInput struct {
	Body struct {
		GameID string `json:"gameId,omitempty" format:"uuid" required:"false" doc:"Game to assign the deck to. Omit to leave the deck unassigned." example:"1b4e28ba-2fa1-11d2-883f-0016d3cca427"`
	}
}

// CreateDeckOutput is the response for a created deck.
type CreateDeckOutput struct {
	Body Deck
}

// ListDecksInput carries no parameters; listing takes no arguments.
type ListDecksInput struct{}

// ListDecksOutput is the response carrying every deck.
type ListDecksOutput struct {
	Body []Deck
}

// Register attaches the deck operations to the API.
func (h *DeckHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-deck",
		Method:        http.MethodPost,
		Path:          "/api/v1/decks",
		Summary:       "Create a deck",
		Description:   "Creates a deck initialized with the 52 cards of a standard deck.",
		Tags:          []string{"deck"},
		DefaultStatus: http.StatusCreated,
	}, h.Create)

	huma.Register(api, huma.Operation{
		OperationID:   "list-decks",
		Method:        http.MethodGet,
		Path:          "/api/v1/decks",
		Summary:       "List decks",
		Tags:          []string{"deck"},
		DefaultStatus: http.StatusOK,
	}, h.List)
}

// Create handles POST /api/v1/decks.
func (h *DeckHandler) Create(ctx context.Context, in *CreateDeckInput) (*CreateDeckOutput, error) {
	var gameID *uuid.UUID
	if in.Body.GameID != "" {
		id, err := uuid.Parse(in.Body.GameID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid game id", err)
		}
		gameID = &id
	}

	d, err := h.svc.Create(ctx, gameID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("game not found")
		}
		return nil, huma.Error500InternalServerError("failed to create deck", err)
	}

	return &CreateDeckOutput{Body: newDeck(d)}, nil
}

// List handles GET /api/v1/decks.
func (h *DeckHandler) List(ctx context.Context, _ *ListDecksInput) (*ListDecksOutput, error) {
	decks, err := h.svc.List(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list decks", err)
	}
	// Built empty rather than nil so an empty listing serializes as [], not null.
	body := make([]Deck, 0, len(decks))
	for _, d := range decks {
		body = append(body, newDeck(d))
	}
	return &ListDecksOutput{Body: body}, nil
}
