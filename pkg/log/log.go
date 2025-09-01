// Package log 提供基于 zerolog 的日志工具，支持 stdout/stderr 和文件输出（lumberjack 轮转）。
package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

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

// initLogger 实际执行一次的初始化函数。
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
	}

	logger = ctx.Timestamp().Logger()

	log.Logger = logger
}

// Logger 返回全局 logger.
func Logger() zerolog.Logger {
	// ensure logger is initialized on first use
	initOnce.Do(initLogger)

	return logger
}
