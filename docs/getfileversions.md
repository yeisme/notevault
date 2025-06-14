# 获取文件版本历史处理流程详解

本文档详细介绍获取文件版本历史的处理流程，并通过多个mermaid图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始获取版本历史] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D[查询文件基本信息]
    D --> E{文件存在?}
    E -->|否| F[返回文件不存在错误]
    E -->|是| G{用户有权限?}
    G -->|否| H[返回权限错误]
    G -->|是| I[查询版本记录]
    I --> J[按版本号排序]
    J --> K[构建版本信息列表]
    K --> L[返回版本历史响应]
```

## 详细步骤分析

### 1. 请求处理与验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Database
    Client->>Handler: GET /api/v1/files/:fileId/versions
    Handler->>Logic: GetFileVersions(req)
    Logic->>Database: 查询文件基本信息
    Database-->>Logic: 返回文件元数据
    Logic->>Logic: 验证用户权限
    Logic->>Database: 查询所有版本记录
    Database-->>Logic: 返回版本列表
    Logic->>Logic: 排序和格式化版本信息
    Logic-->>Handler: 返回版本历史响应
```

### 2. 版本查询处理

```mermaid
flowchart TD
    A[查询版本记录] --> B[构建查询条件]
    B --> C[执行数据库查询]
    C --> D{查询成功?}
    D -->|否| E[记录查询错误]
    D -->|是| F[获取版本列表]
    F --> G[按版本号降序排序]
    G --> H[遍历版本记录]
    H --> I[构建版本信息对象]
    I --> J{还有版本?}
    J -->|是| H
    J -->|否| K[返回完整版本列表]
    E --> L[返回查询失败错误]
```

### 3. 版本信息构建流程

```mermaid
sequenceDiagram
    participant Logic
    participant VersionRecord
    participant VersionInfo
    Logic->>VersionRecord: 获取版本数据
    VersionRecord-->>Logic: 返回版本详细信息
    Logic->>VersionInfo: 创建版本信息对象
    Note over VersionInfo: 包含版本号、大小、时间等
    VersionInfo-->>Logic: 返回格式化的版本信息
    Logic->>Logic: 添加到版本列表
    Logic->>Logic: 重复处理下一个版本
```

## 数据查询示意图

```mermaid
erDiagram
    FILE ||--o{ FILE_VERSION : has_versions
    
    FILE {
        string file_id PK
        string user_id
        string file_name
        int current_version
        int64 created_at
        int64 updated_at
    }
    
    FILE_VERSION {
        string version_id PK
        string file_id FK
        int version_number
        int64 size
        string path
        string content_type
        int64 created_at
        int16 status
        string commit_message
    }
```

## 版本状态管理

```mermaid
stateDiagram-v2
    [*] --> Active: 版本创建
    Active --> Replaced: 新版本上传
    Active --> Deleted: 版本删除
    Replaced --> Deleted: 清理旧版本
    Deleted --> [*]: 物理删除
    
    state Active {
        [*] --> Current: 当前版本
        [*] --> Historical: 历史版本
        Current --> Historical: 新版本产生
    }
```

## 响应结构说明

```mermaid
classDiagram
    class GetFileVersionsRequest {
        +string FileID
    }
    
    class GetFileVersionsResponse {
        +string FileID
        +[]FileVersionInfo Versions
    }
    
    class FileVersionInfo {
        +int Version
        +int64 Size
        +int64 CreatedAt
        +string ContentType
        +string CommitMessage
    }
    
    GetFileVersionsRequest --> GetFileVersionsResponse: 处理后返回
    GetFileVersionsResponse --> FileVersionInfo: 包含版本信息列表
```

## 版本排序逻辑

```mermaid
flowchart TD
    A[获取版本列表] --> B[按版本号排序]
    B --> C{排序方式}
    C -->|降序| D[最新版本在前]
    C -->|升序| E[最老版本在前]
    D --> F[Version 3]
    F --> G[Version 2] 
    G --> H[Version 1]
    E --> I[Version 1]
    I --> J[Version 2]
    J --> K[Version 3]
```

## 使用示例

### 请求示例

```http
GET /api/v1/files/abc123def456/versions HTTP/1.1
Host: localhost:8080
Authorization: Bearer <jwt-token>
```

### 响应示例

```json
{
    "fileId": "abc123def456",
    "versions": [
        {
            "version": 3,
            "size": 2048576,
            "createdAt": 1634567890,
            "contentType": "application/pdf",
            "commitMessage": "更新文件元数据"
        },
        {
            "version": 2,
            "size": 2045678,
            "createdAt": 1634567800,
            "contentType": "application/pdf", 
            "commitMessage": "修复文档内容错误"
        },
        {
            "version": 1,
            "size": 2041234,
            "createdAt": 1634567700,
            "contentType": "application/pdf",
            "commitMessage": "初始版本上传"
        }
    ]
}
```

## 关键特性说明

### 1. 权限控制

- 只有文件所有者可以查看版本历史
- 支持基于JWT的身份验证
- 详细的权限检查日志

### 2. 数据完整性

- 查询时过滤已删除的版本
- 确保版本号的连续性
- 维护版本创建时间顺序

### 3. 性能优化

- 使用数据库索引加速查询
- 按需加载版本信息
- 合理的查询字段选择

### 4. 错误处理

- 文件不存在时的友好提示
- 权限不足时的明确错误信息
- 数据库查询失败的错误记录

## 版本信息字段说明

| 字段名        | 类型   | 说明                       |
| ------------- | ------ | -------------------------- |
| version       | int    | 版本号，从1开始递增        |
| size          | int64  | 该版本文件的大小（字节）   |
| createdAt     | int64  | 版本创建时间（Unix时间戳） |
| contentType   | string | 文件的MIME类型             |
| commitMessage | string | 版本提交信息（可选）       |

## 扩展功能

### 1. 版本过滤

```mermaid
flowchart TD
    A[版本过滤选项] --> B{按状态过滤}
    B -->|活跃版本| C[status = 0]
    B -->|已替换版本| D[status = 1] 
    B -->|已删除版本| E[status = 2]
    C --> F[返回过滤结果]
    D --> F
    E --> F
```

### 2. 分页支持

```mermaid
sequenceDiagram
    participant Client
    participant Logic
    participant Database
    Client->>Logic: 请求版本历史(page=1, size=10)
    Logic->>Database: 查询指定范围的版本
    Database-->>Logic: 返回分页版本数据
    Logic->>Logic: 计算总页数和总数量
    Logic-->>Client: 返回分页版本响应
```

### 3. 版本差异预览

```mermaid
flowchart TD
    A[版本列表响应] --> B[包含差异信息]
    B --> C[与前一版本的差异]
    C --> D[文件大小变化]
    D --> E[内容修改摘要]
    E --> F[修改时间间隔]
```

## 性能考虑

1. **索引优化**：在 `file_id` 和 `version_number` 上建立复合索引
2. **查询限制**：合理限制返回的版本数量
3. **缓存策略**：对热点文件的版本信息进行缓存
4. **异步加载**：大文件的版本历史可以异步加载详细信息

## 监控指标

- 版本历史查询频率
- 平均版本数量统计
- 查询响应时间
- 错误率统计

整个版本历史查询流程设计简洁高效，为用户提供了清晰的文件演进历史，有助于文件管理和版本控制。
