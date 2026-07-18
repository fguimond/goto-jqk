// Package store provides persistence for domain entities. The current
// implementation keeps everything in memory.
package store

import (
	"errors"

	"github.com/google/uuid"

	"github.com/fguimond/goto-jqk/internal/model"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// GameStore is the persistence contract for games.
type GameStore interface {
	Create(g *model.Game) error
	Delete(id uuid.UUID) error
}
