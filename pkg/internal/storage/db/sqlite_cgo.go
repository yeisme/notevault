//go:build !no_sqlite && cgo

package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/configs"
)

// createSQLiteDialector 创建SQLite dialector (CGo版本).
func createSQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}

// 注册SQLite dialector工厂函数 (CGo版本).
func init() {
	RegisterDialectorFactory(configs.SQLite, createSQLiteDialector)
}
