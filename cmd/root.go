package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strava",
	Short: "Strava is a CLI tool to interact with Strava.",
	Long:  `Strava is a CLI tool to interact with Strava.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(timestampCmd)
}
