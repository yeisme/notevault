package configs

import (
	"github.com/spf13/viper"
)

const (
	DefaultLogEnableFile = true                 // 是否启用文件日志
	DefaultLogFilePath   = "logs/notevault.log" // 日志文件路径
	DefaultLogMaxSize    = 100                  // 日志文件最大尺寸（MB）
	DefaultLogMaxBackups = 7                    // 日志文件最大备份数量
	DefaultLogMaxAge     = 28                   // 日志文件最大保存天数
	DefaultLogCompress   = true                 // 是否启用日志文件压缩
	DefaultLogLevel      = "info"               // 日志级别
)

type (
	// LogConfig 日志相关配置.
	LogConfig struct {
		EnableFile bool   `mapstructure:"enable_file"`
		FilePath   string `mapstructure:"file_path"`
		MaxSize    int    `mapstructure:"max_size_mb"`
		MaxBackups int    `mapstructure:"max_backups"`
		MaxAge     int    `mapstructure:"max_age_days"`
		Compress   bool   `mapstructure:"compress"`
		Level      string `mapstructure:"level"`
	}
)

func (l *LogConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("log.enable_file", DefaultLogEnableFile)
	v.SetDefault("log.file_path", DefaultLogFilePath)
	v.SetDefault("log.max_size_mb", DefaultLogMaxSize)
	v.SetDefault("log.max_backups", DefaultLogMaxBackups)
	v.SetDefault("log.max_age_days", DefaultLogMaxAge)
	v.SetDefault("log.compress", DefaultLogCompress)
	v.SetDefault("log.level", DefaultLogLevel)
}
