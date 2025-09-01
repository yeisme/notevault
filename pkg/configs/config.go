// Package configs 管理应用程序配置，包括数据库、存储和队列的配置信息。
package configs

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type (
	// AppConfig 全局应用程序配置
	AppConfig struct {
		DB     DBConfig     `mapstructure:"db"`
		S3     S3Config     `mapstructure:"s3"`     // S3Config 对象存储配置
		MQ     MQConfig     `mapstructure:"mq"`     // MQConfig 消息队列配置
		Server ServerConfig `mapstructure:"server"` // ServerConfig 其它服务器配置，日志级别、服务器端口等
	}
)

var (
	// GlobalConfig 全局配置实例
	GlobalConfig AppConfig
	// AppViper 全局 Viper 实例
	AppViper *viper.Viper
)

// InitConfig 加载应用程序配置，支持多种格式(yaml、json、toml、dotenv)并启用热重载
func InitConfig(path string) error {
	AppViper := viper.New()

	AppViper.SetConfigName("config")

	AppViper.SetConfigType("yaml")
	AppViper.SetConfigType("yml")
	AppViper.SetConfigType("json")
	AppViper.SetConfigType("toml")
	AppViper.SetConfigType("dotenv")

	AppViper.AutomaticEnv()
	AppViper.SetEnvPrefix("NOTEVAULT")

	AppViper.AddConfigPath(path)
	AppViper.AddConfigPath(path + "./configs")

	// 设置默认值
	setAllDefaults(AppViper)

	// 读取配置
	if err := AppViper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// 解析到全局配置
	if err := AppViper.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	reloadConfigs(AppViper, GlobalConfig.Server.ReloadConfig)

	return nil
}

// setAllDefaults 设置所有配置的默认值
func setAllDefaults(v *viper.Viper) {
	var serverConfig ServerConfig
	var dbConfig DBConfig
	var s3Config S3Config
	var mqConfig MQConfig

	serverConfig.setDefaults(v)
	dbConfig.setDefaults(v)
	s3Config.setDefaults(v)
	mqConfig.setDefaults(v)
}

func reloadConfigs(v *viper.Viper, isHotReload bool) {
	if !isHotReload {
		return
	}
	// 启用配置热重载
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		fmt.Println("Reloading configuration...")
		if err := v.Unmarshal(&GlobalConfig); err != nil {
			fmt.Printf("Error reloading config: %v\n", err)
		}
	})
	v.WatchConfig()
}

// GetConfig 返回全局配置实例
func GetConfig() *AppConfig {
	return &GlobalConfig
}
