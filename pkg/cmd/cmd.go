// Package cmd contains the command line applications for the project.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yeisme/notevault/pkg/app"
	"github.com/yeisme/notevault/pkg/configs"
	"github.com/yeisme/notevault/pkg/log"
)

var (
	rootCmd = &cobra.Command{
		Use:   "notevault",
		Short: "A command line tool for managing notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			restartChan := make(chan struct{}, 1)

			// 设置配置重新加载回调
			configs.SetReloadCallback(func() {
				select {
				case restartChan <- struct{}{}:
				default:
					// 如果通道已满，忽略这次重新启动信号
				}
			})

			for {
				applications := app.NewApp(configPath)

				// 启动应用并监听重新启动信号
				errChan := make(chan error, 1)
				go func() {
					errChan <- applications.Run()
				}()

				select {
				case err := <-errChan: // 应用错误处理
					// 应用正常退出或出错
					if err != nil {
						return err
					}
					return nil
				case <-restartChan: // 接收重新启动信号
					// 收到重新启动信号，停止当前应用
					fmt.Println("Configuration changed, restarting server...")
					applications.Shutdown()
					// 继续循环，重新创建应用
				}
			}
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 初始化配置
			if err := configs.InitConfig(configPath); err != nil {
				fmt.Printf("Error initializing config: %v\n", err)
				os.Exit(1)
			}
			log.Init()
		},
	}
)

// Execute runs the root command.
func Execute() error {
	// 设置持久化全局命令行标志
	setupFlags()
	// 绑定命令行标志到 AppViper
	bindFlagsToViper(rootCmd)
	// 注册子命令
	registerConfigsCommands()
	registerDBCommands()
	registerKVCommands()
	registerMQCommands()

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
	// Prefer env vars for default config path when provided
	defaultCfg := os.Getenv("NOTEVAULT_CONFIG")
	if defaultCfg == "" {
		defaultCfg = os.Getenv("NOTEVAULT_CONFIG_PATH")
	}

	if defaultCfg == "" {
		defaultCfg = "."
	}

	rootCmd.PersistentFlags().StringVarP(
		&configPath, "config", "c", defaultCfg, "config file or directory (can be set via NOTEVAULT_CONFIG or NOTEVAULT_CONFIG_PATH)")

	// ServerConfig 相关标志
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", configs.DefaultPort, "server port")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", configs.DefaultHost, "server host")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", configs.DefaultLogLevel, "log level")
	rootCmd.PersistentFlags().BoolVarP(&reload, "reload-config", "r", configs.DefaultReloadConfig, "enable config hot reload")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", configs.DefaultDebug, "enable debug mode")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", configs.DefaultTimeout, "timeout in seconds")
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
