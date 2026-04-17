package cmd

import (
	"log"

	"email-bot/internal/tui"

	"github.com/spf13/cobra"
)

var apiAddress string

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the email-bot TUI dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.StartTUI(apiAddress); err != nil {
			log.Fatalf("Error running TUI: %v", err)
		}
	},
}

func init() {
	tuiCmd.Flags().StringVarP(&apiAddress, "api", "a", "127.0.0.1:8080", "API address of the daemon")
	rootCmd.AddCommand(tuiCmd)
}
