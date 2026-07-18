package cli

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/fguimond/goto-jqk/internal/api"
	"github.com/fguimond/goto-jqk/internal/logging"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the REST API server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return run()
		},
	}

	cmd.Flags().String("addr", ":8080", "address the server listens on")
	_ = viper.BindPFlag("addr", cmd.Flags().Lookup("addr"))

	return cmd
}

func run() error {
	logger := logging.New(viper.GetString("log-level"))
	slog.SetDefault(logger)

	addr := viper.GetString("addr")
	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server stopped")
	return nil
}
