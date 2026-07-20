package service

import "log/slog"

// testLogger returns a logger that drops everything, so the operation logs the
// services emit do not clutter test output.
func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
