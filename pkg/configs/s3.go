package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	DefaultS3Endpoint        = "localhost:9000" // 默认S3端点
	DefaultS3AccessKeyID     = "minioadmin"     // 默认访问密钥ID
	DefaultS3SecretAccessKey = "minioadmin"     // 默认秘密访问密钥
	DefaultS3UseSSL          = false            // 默认是否使用SSL
	DefaultS3BucketName      = "notevault"      // 默认单个存储桶名称（向后兼容，转换为切片）
	DefaultS3Region          = "us-east-1"      // 默认区域
)

// S3Config MinIO S3存储配置.
type S3Config struct {
	Endpoint        string   `mapstructure:"endpoint"          rule:"hostname_port"`
	AccessKeyID     string   `mapstructure:"access_key_id"`
	SecretAccessKey string   `mapstructure:"secret_access_key"`
	UseSSL          bool     `mapstructure:"use_ssl"`
	Buckets         []string `mapstructure:"buckets"`     // 支持多个 bucket
	Region          string   `mapstructure:"region"`
}

// GetEndpointURL 获取完整的端点URL.
func (c *S3Config) GetEndpointURL() string {
	scheme := "http"
	if c.UseSSL {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, c.Endpoint)
}


// setDefaults 设置 S3 配置的默认值.
func (c *S3Config) setDefaults(v *viper.Viper) {
	v.SetDefault("storage.endpoint", DefaultS3Endpoint)
	v.SetDefault("storage.access_key_id", DefaultS3AccessKeyID)
	v.SetDefault("storage.secret_access_key", DefaultS3SecretAccessKey)
	v.SetDefault("storage.use_ssl", DefaultS3UseSSL)
	v.SetDefault("storage.bucket_name", DefaultS3BucketName)
	v.SetDefault("storage.buckets", []string{DefaultS3BucketName})
	v.SetDefault("storage.region", DefaultS3Region)
}
