package service

import "log/slog"

// Entity and operation names stamped on every mutation log record. They are
// constants rather than literals at each call site so a record can always be
// filtered on (entity, operation) — msg stays human-readable prose and is never
// the thing queried on.
//
// Operations are only ever create, update or delete: a shuffle, a deal and a
// deck assignment are all updates to the resource they act on, distinguished by
// msg and the attributes that follow.
const (
	entityGame   = "game"
	entityDeck   = "deck"
	entityPlayer = "player"

	opCreate = "create"
	opUpdate = "update"
	opDelete = "delete"
)

// opLogger derives a logger that tags every record with the entity being
// changed and the kind of change, so the success and failure records of one
// operation carry identical labels.
func opLogger(l *slog.Logger, entity, operation string) *slog.Logger {
	return l.With(
		slog.String("entity", entity),
		slog.String("operation", operation),
	)
}
