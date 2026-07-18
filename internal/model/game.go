// Package model contains the core domain types.
package model

import "github.com/google/uuid"

// Game is the core domain entity. For now it only carries a name.
type Game struct {
	ID   uuid.UUID
	Name string
}
