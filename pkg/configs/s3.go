package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

// S3Config MinIO S3存储配置
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
}

// setDefaults 设置 S3 配置的默认值
func (c *S3Config) setDefaults(v *viper.Viper) {
	v.SetDefault("storage.endpoint", "localhost:9000")
	v.SetDefault("storage.access_key_id", "minioadmin")
	v.SetDefault("storage.secret_access_key", "minioadmin")
	v.SetDefault("storage.use_ssl", false)
	v.SetDefault("storage.bucket_name", "notevault")
	v.SetDefault("storage.region", "us-east-1")
}

// GetEndpointURL 获取完整的端点URL
func (c *S3Config) GetEndpointURL() string {
	scheme := "http"
	if c.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Endpoint)
}
