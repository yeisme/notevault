package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	mq "github.com/yeisme/notevault/pkg/internal/storage/mq"
)

var (
	mqCmd = &cobra.Command{
		Use:     "mq",
		Short:   "Message queue related commands",
		Aliases: []string{"messagequeue"},
	}

	mqListCmd = &cobra.Command{
		Use:     "list",
		Short:   "list all registered mq types",
		Aliases: []string{"ls", "l"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Registered mq types:")
			for _, t := range mq.GetRegisteredMQTypes() {
				fmt.Fprintln(cmd.OutOrStdout(), "   - "+string(t))
			}
		},
	}
)

// registerMQCommands 注册 MQ 相关命令.
func registerMQCommands() {
	rootCmd.AddCommand(mqCmd)
	mqCmd.AddCommand(mqListCmd)
}
