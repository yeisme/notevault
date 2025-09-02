// Package s3 处理S3存储操作.
package s3

import (
	"context"
	"fmt"
	"net/url"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yeisme/notevault/pkg/configs"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Client 包装 MinIO 客户端.
type Client struct {
	*minio.Client
}

// New 初始化 MinIO 客户端，若 bucket 不存在则尝试创建.
func New(ctx context.Context, cfg *configs.S3Config) (*Client, error) {
	endpoint := cfg.Endpoint
	// 允许用户传完整 schema endpoint（http:// 或 https://）
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		endpoint = u.Host
		if u.Scheme == "https" {
			cfg.UseSSL = true
		}
	}

	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// ensure all buckets
	for i, bkt := range cfg.Buckets {
		if bkt == "" {
			continue
		}

		exists, err := cli.BucketExists(ctx, bkt)
		if err != nil {
			return nil, fmt.Errorf("check bucket %s: %w", bkt, err)
		}

		if !exists {
			if err := cli.MakeBucket(ctx, bkt, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
				return nil, fmt.Errorf("create bucket %s: %w", bkt, err)
			}

			nlog.Logger().Info().Str("bucket", bkt).Msgf("bucket %d created", i)
		}
	}

	nlog.Logger().Info().Str("endpoint", cfg.Endpoint).Int("bucket_count", len(cfg.Buckets)).Msg("s3 connected")

	return &Client{Client: cli}, nil
}
