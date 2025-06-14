# 批量删除文件处理流程详解

本文档详细介绍批量删除文件的处理流程，并通过多个mermaid图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始批量删除] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D{文件ID列表为空?}
    D -->|是| E[返回无文件删除消息]
    D -->|否| F[初始化响应结构]
    F --> G[遍历文件ID列表]
    G --> H[处理单个文件删除]
    H --> I{删除成功?}
    I -->|是| J[添加到成功列表]
    I -->|否| K[添加到失败列表]
    J --> L{还有文件?}
    K --> L
    L -->|是| G
    L -->|否| M[构建批量删除响应]
    M --> N[返回批量删除结果]
```

## 详细步骤分析

### 1. 请求处理与身份验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Context
    Client->>Handler: POST /api/v1/files/batch/delete
    Handler->>Logic: BatchDeleteFiles(req)
    Logic->>Context: 获取userId
    alt 获取成功
        Context-->>Logic: 返回userId
    else 获取失败
        Logic-->>Logic: 使用默认用户ID
    end
    Logic-->>Handler: 继续处理
```

### 2. 单个文件删除处理流程

```mermaid
flowchart TD
    A[处理单个文件] --> B[查询文件信息]
    B --> C{文件存在?}
    C -->|否| D[记录删除失败]
    C -->|是| E{用户有权限?}
    E -->|否| F[记录权限错误]
    E -->|是| G[开始数据库事务]
    G --> H[软删除文件记录]
    H --> I[设置trashed_at时间戳]
    I --> J[软删除相关版本]
    J --> K[删除标签关联]
    K --> L{事务成功?}
    L -->|是| M[记录删除成功]
    L -->|否| N[记录删除失败]
    D --> O[继续下一个文件]
    F --> O
    M --> O
    N --> O
```

### 3. 数据库操作详细流程

```mermaid
sequenceDiagram
    participant Logic
    participant Database
    participant Transaction
    Logic->>Database: 查询文件信息
    Database-->>Logic: 返回文件元数据
    Logic->>Transaction: 开始事务
    Transaction->>Database: 更新文件状态为待删除(status=3)
    Database-->>Transaction: 更新成功
    Transaction->>Database: 设置deleted_at和trashed_at时间戳
    Database-->>Transaction: 更新成功
    Transaction->>Database: 软删除相关版本记录
    Database-->>Transaction: 更新成功
    Transaction->>Database: 物理删除标签关联记录
    Database-->>Transaction: 删除成功
    Transaction-->>Logic: 提交事务
    Logic-->>Logic: 记录操作结果
```

### 4. 错误处理与结果统计

```mermaid
flowchart TD
    A[开始统计] --> B[初始化计数器]
    B --> C[遍历处理结果]
    C --> D{处理成功?}
    D -->|是| E[成功计数+1]
    D -->|否| F[失败计数+1]
    E --> G[添加到成功列表]
    F --> H[添加到失败列表]
    G --> I{还有结果?}
    H --> I
    I -->|是| C
    I -->|否| J[生成汇总消息]
    J --> K[返回批量删除响应]
```

## 数据变更示意图

```mermaid
stateDiagram-v2
    [*] --> Normal: 文件创建
    Normal --> Trashed: 批量删除操作
    Trashed --> PermanentlyDeleted: 清理任务
    
    state Normal {
        [*] --> Active
        Active --> Active: 文件操作
    }
    
    state Trashed {
        [*] --> MarkedForDeletion
        MarkedForDeletion: status=3, trashed_at设置
        MarkedForDeletion: deleted_at设置
    }
    
    state PermanentlyDeleted {
        [*] --> RemovedFromOSS
        RemovedFromOSS: OSS文件删除
    }
```

## 响应结构说明

```mermaid
classDiagram
    class BatchDeleteFilesRequest {
        +[]string FileIDs
        +*int VersionNumber
    }
    
    class BatchDeleteFilesResponse {
        +[]string Succeeded
        +[]string Failed
        +string Message
    }
    
    BatchDeleteFilesRequest --> BatchDeleteFilesResponse: 处理后返回
```

## 关键特性说明

### 1. 原子性保证
- 每个文件的删除操作使用独立的数据库事务
- 单个文件删除失败不会影响其他文件的处理
- 提供详细的成功/失败文件列表

### 2. 软删除策略
- 文件记录标记为待删除状态(status=3)
- 设置 `deleted_at` 和 `trashed_at` 时间戳
- OSS文件保留，便于后续恢复

### 3. 关联数据处理
- 文件版本记录同步软删除
- 标签关联记录物理删除
- 保持数据一致性

### 4. 错误容错机制
- 单个文件处理失败时记录错误但继续处理
- 版本删除失败不阻断主流程
- 标签关联删除失败仅记录日志

## 使用示例

### 请求示例
```json
{
    "fileIds": [
        "abc123def456",
        "xyz789uvw012",
        "mno345pqr678"
    ]
}
```

### 响应示例
```json
{
    "succeeded": [
        "abc123def456",
        "mno345pqr678"
    ],
    "failed": [
        "xyz789uvw012"
    ],
    "message": "Batch move to trash completed: 2 succeeded, 1 failed"
}
```

## 性能考虑

1. **批量处理效率**：逐个处理文件，避免大事务锁定
2. **数据库连接**：复用查询构建器，减少连接开销
3. **错误隔离**：单个失败不影响整体处理
4. **日志记录**：详细记录操作过程便于问题排查

整个批量删除流程设计注重稳定性和用户体验，确保即使在部分文件处理失败的情况下也能提供有用的反馈信息。
