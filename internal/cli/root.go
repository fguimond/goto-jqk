// Package cli defines the cobra/viper command-line interface, structured in the
// same style as kubectl (a root command with subcommands).
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// NewRootCmd builds the root command and attaches all subcommands. cobra
// automatically provides the `help` and `completion` commands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "goto-jqk",
		Short: "goto-jqk is a REST API server",
		Long: "goto-jqk is a REST API server scaffolded with cobra/viper, " +
			"a huma-generated OpenAPI schema, and structured slog logging.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.goto-jqk.yaml)")
	root.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")

	_ = viper.BindPFlag("log-level", root.PersistentFlags().Lookup("log-level"))

	cobra.OnInitialize(initConfig)

	root.AddCommand(newRunCmd())
	root.AddCommand(newVersionCmd())

	return root
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".goto-jqk")
	}

	viper.SetEnvPrefix("GOTO_JQK")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// A missing config file is not an error; env vars and flags still apply.
	_ = viper.ReadInConfig()
}
