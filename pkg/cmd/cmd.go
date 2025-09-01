// Package cmd contains the command line applications for the project.
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "notevault",
		Short: "A command line tool for managing notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			
			return nil
		},
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
