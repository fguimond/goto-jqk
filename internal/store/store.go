// Package store provides persistence for domain entities. The current
// implementation keeps everything in memory.
//
// Note: consumers define the storage interfaces they need at their own point
// of use (see service.GameStore); this package only exports concrete
// implementations and the sentinel errors they can return.
package store

import "errors"

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")
