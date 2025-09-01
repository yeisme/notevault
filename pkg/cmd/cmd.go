// Package cmd contains the command line applications for the project.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yeisme/notevault/pkg/configs"
)

var (
	rootCmd = &cobra.Command{
		Use:   "notevault",
		Short: "A command line tool for managing notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 这里可以添加应用程序的主要逻辑

			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 初始化配置
			if err := configs.InitConfig(configPath); err != nil {
				fmt.Printf("Error initializing config: %v\n", err)
				os.Exit(1)
			}

			// 绑定命令行标志到 AppViper
			bindFlagsToViper(cmd)
		},
	}
)

// Execute runs the root command.
func Execute() error {
	setupFlags()

	// 注册子命令
	registerConfigsCommands()

	return rootCmd.Execute()
}

var (
	configPath string
	port       int
	host       string
	logLevel   string
	reload     bool
	debug      bool
	timeout    int
)

// setupFlags 设置命令行标志并绑定到 viper.
func setupFlags() {
	rootCmd.PersistentFlags().StringVarP(
		&configPath, "config", "c", ".", "config file (default discovered in current directory or ./configs)")

	// ServerConfig 相关标志
	rootCmd.PersistentFlags().IntVarP(
		&port, "port", "p", configs.DefaultPort, "server port")
	rootCmd.PersistentFlags().StringVarP(
		&host, "host", "H", configs.DefaultHost, "server host")
	rootCmd.PersistentFlags().StringVarP(
		&logLevel, "log-level", "l", configs.DefaultLogLevel, "log level")
	rootCmd.PersistentFlags().BoolVarP(
		&reload, "reload-config", "r", configs.DefaultReloadConfig, "enable config hot reload")
	rootCmd.PersistentFlags().BoolVarP(
		&debug, "debug", "d", configs.DefaultDebug, "enable debug mode")
	rootCmd.PersistentFlags().IntVarP(
		&timeout, "timeout", "t", configs.DefaultTimeout, "timeout in seconds")
}

// bindFlagsToViper 将命令行标志绑定到 AppViper.
func bindFlagsToViper(cmd *cobra.Command) {
	appViper := configs.GetViper()
	if appViper == nil {
		return
	}

	appViper.BindPFlag("server.port", cmd.PersistentFlags().Lookup("port"))
	appViper.BindPFlag("server.host", cmd.PersistentFlags().Lookup("host"))
	appViper.BindPFlag("server.log_level", cmd.PersistentFlags().Lookup("log-level"))
	appViper.BindPFlag("server.reload_config", cmd.PersistentFlags().Lookup("reload-config"))
	appViper.BindPFlag("server.debug", cmd.PersistentFlags().Lookup("debug"))
	appViper.BindPFlag("server.timeout", cmd.PersistentFlags().Lookup("timeout"))
}
