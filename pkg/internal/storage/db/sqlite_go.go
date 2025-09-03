//go:build !no_sqlite && !cgo

package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/configs"
)

// createSQLiteDialector 创建SQLite dialector.
func createSQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}

// 注册SQLite dialector工厂函数.
func init() {
	RegisterDialectorFactory(configs.SQLite, createSQLiteDialector)
}
