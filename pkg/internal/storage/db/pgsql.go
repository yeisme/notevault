//go:build !no_postgres

package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/configs"
)

// createPostgresDialector 创建PostgreSQL dialector.
func createPostgresDialector(dsn string) gorm.Dialector {
	return postgres.Open(dsn)
}

// RegisterPostgresDialector 注册PostgreSQL dialector工厂函数.
func RegisterPostgresDialector() {
	RegisterDialectorFactory(configs.PostgreSQL, createPostgresDialector)
}
