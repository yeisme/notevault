// Package configs 管理应用程序配置，包括数据库、存储和队列的配置信息.
// configs 包支持多种配置格式（YAML、JSON、TOML、dotenv）并启用热重载.
//
// Example:
//
//	import "path/to/configs"
//
//	err := configs.InitConfig("./")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	config := configs.GetConfig()
//	fmt.Println(config.Server.Port)
//
// Example accessing DB config:
//
//	config := configs.GetConfig()
//	dbConfig := config.DB
//	dsn := dbConfig.GetDSN()
//	fmt.Println("DSN:", dsn)
//
// Example accessing S3 config:
//
//	config := configs.GetConfig()
//	s3Config := config.S3
//	endpoint := s3Config.GetEndpointURL()
//	fmt.Println("S3 Endpoint:", endpoint)
//
// Example accessing MQ config:
//
//	config := configs.GetConfig()
//	mqConfig := config.MQ
//	mqType := mqConfig.GetMQType()
//	fmt.Println("MQ Type:", mqType)
//
// Example accessing Server config:
//
//	config := configs.GetConfig()
//	serverConfig := config.Server
//	timeout := serverConfig.GetTimeoutDuration()
//	fmt.Println("Timeout:", timeout)
package configs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/yeisme/notevault/pkg/rule"
)

const (
	AppVersion = "1.0.0"
)

type (
	// AppConfig 全局应用程序配置.
	AppConfig struct {
		DB      DBConfig      `mapstructure:"db"`      // DBConfig 数据库配置
		S3      S3Config      `mapstructure:"s3"`      // S3Config 对象存储配置
		MQ      MQConfig      `mapstructure:"mq"`      // MQConfig 消息队列配置
		KV      KVConfig      `mapstructure:"kv"`      // KVConfig 键值存储配置
		Server  ServerConfig  `mapstructure:"server"`  // ServerConfig 其它服务器配置，日志级别、服务器端口等
		Log     LogConfig     `mapstructure:"log"`     // LogConfig 日志相关配置
		Tracing TracingConfig `mapstructure:"tracing"` // TracingConfig 分布式追踪配置
		Metrics MetricsConfig `mapstructure:"metrics"` // MetricsConfig 监控指标配置
	}
)

var (
	// globalConfig 全局配置实例.
	globalConfig AppConfig
	// appViper 全局 Viper 实例.
	appViper *viper.Viper
	// onReloadCallback 配置重新加载时的回调函数.
	onReloadCallback func()
)

// InitConfig 加载应用程序配置，支持多种格式(yaml、json、toml、dotenv)并启用热重载.
func InitConfig(path string) error {
	appViper = viper.New()
	// 设置默认值
	setAllDefaults(appViper)

	// 检查path是否是文件
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		// 是文件，使用SetConfigFile，Viper会自动检测类型
		appViper.SetConfigFile(path)
	} else {
		// 是目录，设置配置名和路径
		appViper.SetConfigName("config")
		appViper.AddConfigPath(path)
		appViper.AddConfigPath(path + "/configs")

		exts := []string{"yaml", "yml", "json", "toml", "env", "dotenv"}

		for _, ext := range exts {
			cfg := filepath.Join(path, "config."+ext)
			if _, err := os.Stat(cfg); err == nil {
				appViper.SetConfigFile(cfg)

				break
			}
		}
	}

	appViper.AutomaticEnv()
	appViper.SetEnvPrefix("NOTEVAULT")

	// 读取配置
	if err := appViper.ReadInConfig(); err != nil {
		// 如果失败，使用默认值（已经设置）
	} else {
		// 如果成功，重新设置默认值根据读取的值
		setAllDefaults(appViper)
	}

	// 解析到全局配置
	if err := appViper.Unmarshal(&globalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := rule.ValidateStruct(&globalConfig); err != nil {
		return fmt.Errorf("failed to validate config:\n %w", err)
	}

	reloadConfigs(appViper, globalConfig.Server.ReloadConfig, nil)

	return nil
}

// GetConfig 返回全局配置实例.
func GetConfig() *AppConfig {
	return &globalConfig
}

// GetViper 返回全局 Viper 实例.
func GetViper() *viper.Viper {
	return appViper
}

// SetReloadCallback 设置配置重新加载时的回调函数.
// 该函数将在配置文件更改并成功重新加载后调用.
// 使 onReloadCallback 可设置为 nil 以禁用回调.
func SetReloadCallback(callback func()) {
	onReloadCallback = callback
}

// setAllDefaults 设置所有配置的默认值.
func setAllDefaults(v *viper.Viper) {
	var (
		serverConfig  ServerConfig
		dbConfig      DBConfig
		s3Config      S3Config
		mqConfig      MQConfig
		kvConfig      KVConfig
		logConfig     LogConfig
		tracingConfig TracingConfig
		metricsConfig MetricsConfig
	)

	serverConfig.setDefaults(v)
	dbConfig.setDefaults(v)
	s3Config.setDefaults(v)
	mqConfig.setDefaults(v)
	kvConfig.setDefaults(v)
	logConfig.setDefaults(v)
	tracingConfig.setDefaults(v)
	metricsConfig.setDefaults(v)
}

func reloadConfigs(v *viper.Viper, isHotReload bool, onReload func()) {
	if !isHotReload {
		return
	}

	onReloadCallback = onReload
	// 启用配置热重载
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		fmt.Println("Reloading configuration...")

		if err := v.Unmarshal(&globalConfig); err != nil {
			fmt.Printf("Error reloading config:\n %v\n", err)

			return
		}

		// 验证重新加载的配置
		if err := rule.ValidateStruct(&globalConfig); err != nil {
			fmt.Printf("Error validating reloaded config:\n %v\n", err)

			return
		}

		fmt.Println("Configuration reloaded successfully")

		// 调用重新加载回调函数
		if onReloadCallback != nil {
			onReloadCallback()
		}
	})
	v.WatchConfig()
}
