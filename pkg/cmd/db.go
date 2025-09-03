package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/notevault/pkg/internal/storage/db"
)

var (
	dbCmd = &cobra.Command{
		Use:   "db",
		Short: "Database related commands",
	}

	dbListCmd = &cobra.Command{
		Use:   "ls",
		Short: "list all registered database types",
		Run: func(cmd *cobra.Command, args []string) {

			fmt.Fprintln(cmd.OutOrStdout(), "Registered database types:")
			for _, dbType := range db.GetRegisteredDBTypes() {
				fmt.Fprintln(cmd.OutOrStdout(), " - "+dbType)
			}
		},
	}
)

// registerDBCommands 注册数据库相关命令.
func registerDBCommands() {
	rootCmd.AddCommand(dbCmd)

	dbCmd.AddCommand(dbListCmd)
}
