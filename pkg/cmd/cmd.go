// Package cmd contains the command line applications for the project.
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/notevault/pkg/configs"
	"gofr.dev/pkg/gofr"
)

var (
	rootCmd = &cobra.Command{
		Use:   "notevault",
		Short: "A command line tool for managing notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := gofr.New()

			if err := configs.LoadConfig(&app.Config); err != nil {
				return err
			}

			app.Run()
			return nil
		},
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
