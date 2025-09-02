package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/notevault/pkg/configs"
)

var (
	// config 子命令.
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "config subcommands",
	}

	// 打印当前使用的配置文件路径.
	pathCmd = &cobra.Command{
		Use:   "path",
		Short: "print the path of the current config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := configs.GetViper()
			if v == nil {
				fmt.Println("config not initialized")

				return nil
			}

			cfg := v.ConfigFileUsed()
			if cfg == "" {
				fmt.Println("no config file used (maybe using defaults or env)")

				return nil
			}

			fmt.Println(cfg)

			return nil
		},
	}

	// 调用 viper 的 Debug 输出.
	debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "print the current config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := configs.GetViper()
			c := configs.GetConfig()
			if v == nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "config not initialized.")

				return nil
			}

			if debug {
				v.Debug()
			}

			// 以 JSON 格式打印当前配置
			b, err := json.MarshalIndent(c, "", "  ")
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "failed to marshal config to JSON:", err)

				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(b))

			return nil
		},
	}
)

// registerConfigsCommands 注册 CLI 子命令.
func registerConfigsCommands() {
	configCmd.AddCommand(pathCmd)
	configCmd.AddCommand(debugCmd)

	rootCmd.AddCommand(configCmd)
}
