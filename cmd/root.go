package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "email-bot",
	Short: "A daemon + TUI email forwarding bot",
	Long:  `email-bot fetches emails from multiple IMAP sources and forwards them to SMTP targets.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags if necessary
}
