# notevault

notevault 核心功能是将文件（多模态数据）存储到对象存储中，并提供 http 等服务来访问这些文件

1. S3 对象存储: 提供高可用性和高可靠性的对象存储服务
2. 消息队列异步解析: 支持异步处理和解析多模态数据
3. 多模态数据元数据管理: 提供对多模态数据的统一管理和访问能力
4. HTTP 服务: 提供高性能的 API 接口

## 快速开始

### 使用 Docker Compose 进行本地开发

1. 确保已安装 Docker 和 Docker Compose

2. 克隆项目并进入目录：

   ```bash
   git clone <repository-url>
   cd notevault
   ```

3. 启动所有服务：

   ```bash
   docker-compose up -d
   ```

   ```bash
   go run cmd/notevault/main.go
   ```

4. 服务启动后，可以访问：

   - NoteVault API: <http://localhost:8080>
   - MinIO Console: <http://localhost:9001> (用户名: minioadmin, 密码: minioadmin)
   - Prometheus: <http://localhost:9090>
   - Grafana: <http://localhost:3000> (用户名: admin, 密码: admin)
   - NATS Monitoring: <http://localhost:8222>

5. 查看日志：

   ```bash
   docker-compose logs -f notevault-app
   ```

6. 停止服务：

   ```bash
   docker-compose down
   ```

## 配置

配置文件位于 `configs/config.yaml`，支持以下组件的配置：

- **服务器配置**: 端口、主机、超时等
- **数据库配置**: PostgreSQL 连接信息
- **S3 存储配置**: MinIO 端点和凭据
- **消息队列配置**: NATS 连接信息
- **日志配置**: 文件日志设置
- **监控配置**: Prometheus 指标
- **追踪配置**: OpenTelemetry 分布式追踪

## 架构

- **PostgreSQL**: 主数据库
- **MinIO**: 对象存储
- **NATS**: 消息队列
- **Prometheus**: 监控指标收集
- **Grafana**: 可视化监控面板

---

## 未来计划：微服务设计与目录结构

NoteVault 拆分为聚焦职责的小型服务，事件驱动解耦：

- api-gateway（或复用上层 apigateway）

  - 统一对外 API，转发到下游 notevault-\* 服务

- notevault-upload

  - 预签名直传、分片/合并、上传策略与配额
  - 事件：file.uploaded（包含 object_key、size、content_type、checksum、tenant、uploader）

- notevault-meta

  - 元数据 CRUD、标签/目录、版本控制、软删除与回收站
  - 事件：file.metadata.updated、file.versioned、file.deleted

- notevault-processor

  - 消费 file.uploaded，做魔数嗅探、基础校验，派发解析任务
  - 事件：file.accepted、file.rejected、file.to_parse

- notevault-parser

  - 实际解析（可扩展多种 handler：markdown、pdf、office、image、audio、video）
  - 事件：file.parsed、file.parse_failed

- notevault-indexer

  - 建立可检索索引（文本倒排、向量索引），并回写 meta
  - 事件：file.indexed

- notevault-cleaner
  - 生命周期与归档策略（冷热分层、过期清理、合规擦除）

TODO

module/notevault/

- cmd/
  - notevault-upload/
  - notevault-meta/
  - notevault-processor/
  - notevault-parser/
  - notevault-indexer/
  - notevault-cleaner/
- pkg/
  - api/、service/、repository/、events/、storage/、auth/
- configs/
- docker/、deploy/
- docs/

当前仓库的单体入口可逐步抽离为上述服务，复用现有 pkg 代码，降低迁移成本。

## 事件模型（NATS JetStream 示例）

- 主题与契约（建议结合 Schema Registry/JSON Schema）：
  - notevault.file.uploaded.v1
  - notevault.file.accepted.v1
  - notevault.file.to_parse.v1
  - notevault.file.parsed.v1
  - notevault.file.indexed.v1
  - notevault.file.metadata.updated.v1
  - notevault.file.deleted.v1

事件通用字段：

- id、occurred_at、tenant、actor、trace_id
- payload：与事件类型相关的业务字段（object_key、version_id、checksum、content_type、size 等）

投递保障：

- At-least-once + 幂等消费（业务 idempotency-key）
- 死信队列/重试策略（指数退避）

## API 设计要点

- 上传

  - POST /files/uploads/presign：获取预签名 URL（支持分片 init/part/complete）
  - POST /files/complete：合并分片，写入元数据，发出 uploaded 事件

- 元数据

  - CRUD：名称、标签、描述、目录、ACL、版本
  - 查询：分页、标签过滤、时间范围、全文/向量检索联动（跨服务聚合）

- 下载与访问控制
  - 预签名下载、范围下载、一次性下载券
  - 租户隔离、细粒度权限（与 usercore 集成）

## 存储与索引

- 对象：MinIO 多桶/前缀（按租户/空间划分），SSE 加密
- 关系：PostgreSQL 存元数据与版本；行级安全（RLS）可选
- 缓存：Redis（加速热元数据、上传状态、分片游标）
- 索引：
  - 倒排索引：可选 Meilisearch/OpenSearch，或内置简化版
  - 向量检索：可选 Qdrant/pgvector/Faiss

## 监控与可观测性

- 指标：上传耗时、解析耗时、事件积压、错误率、存储用量
- 追踪：跨服务 trace（ingress -> upload -> processor -> parser -> indexer）
- 日志：结构化日志 + 关联 trace_id

## 渐进式迁移计划

1. 保持现有 REST 接口不变，引入事件发出（uploaded/indexed）
2. 抽离 processor 与 parser 为独立消费服务
3. 引入索引服务，提供搜索聚合 API
4. 最后拆分 upload 与 meta，完成完整微服务化

## 安全与合规

- 访问控制：租户/空间/目录三级策略、最小权限原则
- 数据加密：传输 HTTPS、存储 SSE；密钥轮换
- 合规：可配置保留策略、审计日志、不可变存储（WORM）
