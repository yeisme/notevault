package svc

import (
	"github.com/minio/minio-go/v7"
	"github.com/yeisme/notevault/internal/config"
	"github.com/yeisme/notevault/pkg/mq"
	"github.com/yeisme/notevault/pkg/storage/oss"
	"github.com/yeisme/notevault/pkg/storage/repository/database"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB
	OSS    *minio.Client
	MQ     *mq.MQClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		DB:     database.GetDB(),
		OSS:    oss.GetOssClient(),
		MQ:     mq.GetMQClient(),
	}
}
