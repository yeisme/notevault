//go:build !no_mysql

package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/configs"
)

// createMySQLDialector 创建MySQL dialector.
func createMySQLDialector(dsn string) gorm.Dialector {
	return mysql.Open(dsn)
}

// 注册MySQL dialector工厂函数.
func init() {
	RegisterDialectorFactory(configs.MySQL, createMySQLDialector)
}
