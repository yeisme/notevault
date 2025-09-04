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
