package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"email-bot/internal/api"
	"email-bot/internal/config"
	"email-bot/internal/core"
	"email-bot/internal/state"

	"github.com/spf13/cobra"
)

var configPath string
var statePath string

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the email-bot daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		st, err := state.NewManager(statePath)
		if err != nil {
			log.Fatalf("Failed to initialize state manager: %v", err)
		}

		logger := core.NewLogManager(1000)

		processor := core.NewProcessor(cfg, st, logger)
		server := api.NewServer(cfg, logger)

		go processor.Start()
		go func() {
			if err := server.Start(); err != nil {
				logger.Errorf("API server failed: %v", err)
			}
		}()

		logger.Info("Daemon is running. Press Ctrl+C to exit.")

		// Wait for termination signal
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		logger.Info("Daemon shutting down.")
	},
}

func init() {
	daemonCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to config file")
	daemonCmd.Flags().StringVarP(&statePath, "state", "s", "state.json", "Path to state file")
	rootCmd.AddCommand(daemonCmd)
}
