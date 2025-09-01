package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

type (
	DBType string
)

const (
	// PostgreSQL 协议.
	PostgreSQL DBType = "postgresql"
	Postgres   DBType = "postgre"
	Pg         DBType = "pg"

	// MySQL 协议.
	MySQL   DBType = "mysql"
	MariaDB DBType = "mariadb"
	// SQLite 协议.
	SQLite DBType = "sqlite"
	// DuckDB 协议.
	DuckDB DBType = "duckdb"
)

const (
	DefaultDatabaseHost     = "localhost" // 默认数据库主机
	DefaultDatabasePort     = 5432        // 默认数据库端口
	DefaultDatabaseUser     = "postgres"  // 默认数据库用户
	DefaultDatabasePassword = ""          // 默认数据库密码
	DefaultDatabaseName     = "notevault" // 默认数据库名称
	DefaultDatabaseSSLMode  = "disable"   // 默认数据库SSL模式
	DefaultMaxOpenConns     = 10          // 默认最大打开连接数
	DefaultMaxIdleConns     = 5           // 默认最大空闲连接数
)

// DBConfig 数据库配置.
type DBConfig struct {
	Type         DBType `mapstructure:"type"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// GetDBType 返回数据库类型的字符串表示.
func (c *DBConfig) GetDBType() string {
	switch c.Type {
	case PostgreSQL, Postgres, Pg:
		return "PostgreSQL"
	case MySQL, MariaDB:
		return "MySQL"
	case SQLite:
		return "SQLite"
	case DuckDB:
		return "DuckDB"
	default:
		return "Unknown"
	}
}

// GetDSN 获取数据库的连接字符串，根据不同的数据库类型返回不同格式的DSN
// 通过构建 dsnMap 映射表来简化代码结构和提高可维护性 (优先使用).
func (c *DBConfig) GetDSN() string {
	var dsnMap = map[DBType]func() string{
		PostgreSQL: c.GetPgSQLDSN,
		Postgres:   c.GetPgSQLDSN,
		Pg:         c.GetPgSQLDSN,
		MySQL:      c.GetMySQLDSN,
		MariaDB:    c.GetMySQLDSN,
		SQLite:     c.GetSQLiteDSN,
		DuckDB:     c.GetDuckDBDSN,
	}

	if fn, ok := dsnMap[c.Type]; ok {
		return fn()
	}

	return ""
}

// GetPgSQLDSN 获取PostgreSQL的DSN.
func (c *DBConfig) GetPgSQLDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// GetMySQLDSN 获取MySQL的DSN.
func (c *DBConfig) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// GetSQLiteDSN 获取SQLite的DSN.
func (c *DBConfig) GetSQLiteDSN() string {
	return fmt.Sprintf("file:%s.db", c.Database)
}

// GetDuckDBDSN 获取DuckDB的DSN.
// DuckDB 使用文件路径作为DSN，例如 file:database.db.
func (c *DBConfig) GetDuckDBDSN() string {
	return fmt.Sprintf("file:%s.db", c.Database)
}

// setDefaults 设置数据库配置的默认值.
func (c *DBConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("database.type", PostgreSQL)
	v.SetDefault("database.host", DefaultDatabaseHost)
	v.SetDefault("database.port", DefaultDatabasePort)
	v.SetDefault("database.user", DefaultDatabaseUser)
	v.SetDefault("database.password", DefaultDatabasePassword)
	v.SetDefault("database.database", DefaultDatabaseName)
	v.SetDefault("database.sslmode", DefaultDatabaseSSLMode)
	v.SetDefault("database.max_open_conns", DefaultMaxOpenConns)
	v.SetDefault("database.max_idle_conns", DefaultMaxIdleConns)
}
