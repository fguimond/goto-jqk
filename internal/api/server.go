// Package api assembles the HTTP server: it builds the router, registers the
// huma OpenAPI operations, wires the service/store layers, and applies
// middleware.
package api

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/fguimond/goto-jqk/internal/handler"
	"github.com/fguimond/goto-jqk/internal/service"
	"github.com/fguimond/goto-jqk/internal/store/memory"
	"github.com/fguimond/goto-jqk/internal/version"
)

// NewHandler builds the fully wired HTTP handler for the application.
//
// huma automatically serves:
//   - the OpenAPI spec at /openapi.json and /openapi.yaml
//   - interactive API documentation at /docs
func NewHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("goto-jqk API", version.Version)
	config.Info.Description = "REST API for goto-jqk."
	api := humago.New(mux, config)

	// Compose the layers: store -> service -> handler.
	gameStore := memory.NewGameStore()
	gameSvc := service.NewGameService(gameStore, logger)

	deckStore := memory.NewDeckStore()
	deckSvc := service.NewDeckService(deckStore, gameStore, logger)
	deckHandler := handler.NewDeckHandler(deckSvc)

	// Players live inside the game aggregate, so their service records them
	// against the game store.
	playerSvc := service.NewPlayerService(gameStore, logger)
	playerHandler := handler.NewPlayerHandler(playerSvc)

	// The game handler also serves the deck-assignment route, so it needs the
	// deck service alongside its own.
	gameHandler := handler.NewGameHandler(gameSvc, deckSvc)

	// Register operations on the API.
	gameHandler.Register(api)
	deckHandler.Register(api)
	playerHandler.Register(api)
	handler.RegisterHealth(api)

	return withLogging(logger, mux)
}
