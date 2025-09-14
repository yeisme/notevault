package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	kv "github.com/yeisme/notevault/pkg/internal/storage/kv"
)

var (
	kvCmd = &cobra.Command{
		Use:     "kv",
		Short:   "Key-Value store related commands",
		Aliases: []string{"keyvalue"},
	}

	kvListCmd = &cobra.Command{
		Use:     "list",
		Short:   "list all registered kv types",
		Aliases: []string{"ls", "l"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Registered kv types:")
			for _, t := range kv.GetRegisteredKVTypes() {
				fmt.Fprintln(cmd.OutOrStdout(), "   - "+string(t))
			}
		},
	}
)

// registerKVCommands 注册 KV 相关命令.
func registerKVCommands() {
	rootCmd.AddCommand(kvCmd)
	kvCmd.AddCommand(kvListCmd)
}
