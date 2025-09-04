// Package log 提供基于 zerolog 的日志工具，支持 stdout/stderr 和文件输出（lumberjack 轮转）.
package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/yeisme/notevault/pkg/configs"
)

var (
	logger   zerolog.Logger
	initOnce sync.Once
)

// Init 初始化全局 logger.
func Init() {
	initOnce.Do(initLogger)
}

// initLogger 实际执行一次的初始化函数.
func initLogger() {
	ctg := configs.GetConfig()
	logCfg := ctg.Log

	// level
	lvl, err := zerolog.ParseLevel(strings.ToLower(logCfg.Level))
	if err != nil {
		fmt.Printf("invalid log level %q, defaulting to info", logCfg.Level)

		lvl = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(lvl)

	// outputs
	var writers []io.Writer

	// always add stderr as default human-friendly output, set TimeFormat to time.Kitchen
	console := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.TimeFormat = time.Kitchen
	})
	writers = append(writers, console)

	if logCfg.EnableFile {
		lj := &lumberjack.Logger{
			Filename:   logCfg.FilePath,
			MaxSize:    logCfg.MaxSize,
			MaxBackups: logCfg.MaxBackups,
			MaxAge:     logCfg.MaxAge,
			Compress:   logCfg.Compress,
		}
		writers = append(writers, lj)
	}

	output := io.MultiWriter(writers...)

	ctx := zerolog.New(output).With()
	if ctg.Server.Debug {
		ctx = ctx.Caller().Stack()

		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	logger = ctx.Timestamp().Logger()

	log.Logger = logger
}

// Logger 返回全局 logger.
func Logger() *zerolog.Logger {
	// ensure logger is initialized on first use
	initOnce.Do(initLogger)

	return &logger
}

// GinWriter 把 Gin 文本行转发为 zerolog 事件.
type GinWriter struct {
	logger *zerolog.Logger
	level  zerolog.Level
}

func NewGinWriter(logger *zerolog.Logger, level zerolog.Level) *GinWriter {
	return &GinWriter{logger: logger, level: level}
}

func (w *GinWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	// 使用指定级别记录（按需可扩展解析 level）
	switch w.level {
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		w.logger.Error().Msg(msg)
	case zerolog.WarnLevel:
		w.logger.Warn().Msg(msg)
	default:
		w.logger.Info().Msg(msg)
	}

	return len(p), nil
}
