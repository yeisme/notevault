# 恢复文件版本处理流程详解

本文档详细介绍恢复文件版本的处理流程，并通过多个mermaid图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始版本恢复] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D[查询文件基本信息]
    D --> E{文件存在?}
    E -->|否| F[返回文件不存在错误]
    E -->|是| G{用户有权限?}
    G -->|否| H[返回权限错误]
    G -->|是| I[查询目标版本]
    I --> J{目标版本存在?}
    J -->|否| K[返回版本不存在错误]
    J -->|是| L{目标版本是当前版本?}
    L -->|是| M[返回无需恢复消息]
    L -->|否| N[开始数据库事务]
    N --> O[复制目标版本内容]
    O --> P[创建新版本记录]
    P --> Q[更新文件当前版本]
    Q --> R{事务成功?}
    R -->|否| S[回滚事务]
    R -->|是| T[返回恢复成功响应]
    S --> U[返回恢复失败错误]
```

## 详细步骤分析

### 1. 请求处理与验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Database
    Client->>Handler: POST /api/v1/files/:fileId/versions/revert
    Handler->>Logic: RevertFileVersion(req)
    Logic->>Database: 查询文件基本信息
    Database-->>Logic: 返回文件元数据
    Logic->>Logic: 验证用户权限
    Logic->>Database: 查询目标版本信息
    Database-->>Logic: 返回版本详细信息
    Logic->>Logic: 验证恢复条件
    Logic-->>Handler: 开始恢复处理
```

### 2. 版本恢复处理流程

```mermaid
flowchart TD
    A[获取目标版本信息] --> B[验证版本有效性]
    B --> C{版本状态正常?}
    C -->|否| D[返回版本不可用错误]
    C -->|是| E[计算新版本号]
    E --> F[准备版本数据复制]
    F --> G[创建恢复版本记录]
    G --> H[更新文件主记录]
    H --> I[更新文件路径引用]
    I --> J[设置恢复标记]
    J --> K[提交所有变更]
```

### 3. 数据库事务处理

```mermaid
sequenceDiagram
    participant Logic
    participant Transaction
    participant FileTable
    participant VersionTable
    Logic->>Transaction: 开始事务
    Transaction->>VersionTable: 查询目标版本详细信息
    VersionTable-->>Transaction: 返回版本数据
    Transaction->>FileTable: 递增当前版本号
    FileTable-->>Transaction: 版本号更新成功
    Transaction->>VersionTable: 创建恢复版本记录
    Note over VersionTable: 复制目标版本的内容和元数据
    VersionTable-->>Transaction: 新版本记录创建成功
    Transaction->>FileTable: 更新文件当前路径
    FileTable-->>Transaction: 路径更新成功
    Transaction->>FileTable: 更新修改时间
    FileTable-->>Transaction: 时间戳更新成功
    Transaction-->>Logic: 提交事务
```

## 版本恢复逻辑示意图

```mermaid
stateDiagram-v2
    [*] --> Version1: 初始版本
    Version1 --> Version2: 文件更新
    Version2 --> Version3: 再次更新
    Version3 --> Version4: 恢复到Version1
    Version4 --> Version5: 恢复到Version2
    
    state Version4 {
        [*] --> RevertedFromV1
        RevertedFromV1: 内容来源于Version1
        RevertedFromV1: 版本号为4
    }
    
    state Version5 {
        [*] --> RevertedFromV2
        RevertedFromV2: 内容来源于Version2
        RevertedFromV2: 版本号为5
    }
```

## 数据复制流程

```mermaid
flowchart TD
    A[目标版本数据] --> B[复制文件内容路径]
    B --> C[复制文件大小信息]
    C --> D[复制内容类型]
    D --> E[设置新版本号]
    E --> F[设置创建时间]
    F --> G[设置提交消息]
    G --> H[标记为恢复版本]
    H --> I[保存新版本记录]
```

## 响应结构说明

```mermaid
classDiagram
    class RevertFileVersionRequest {
        +string FileID
        +int Version
        +string CommitMessage
    }
    
    class RevertFileVersionResponse {
        +FileMetadata Metadata
        +string Message
    }
    
    class FileMetadata {
        +string FileID
        +string FileName
        +int Version
        +int64 Size
        +string ContentType
        +int64 UpdatedAt
        +string CommitMessage
    }
    
    RevertFileVersionRequest --> RevertFileVersionResponse: 处理后返回
    RevertFileVersionResponse --> FileMetadata: 包含恢复后的元数据
```

## 版本历史变化

```mermaid
gantt
    title 文件版本时间线
    dateFormat X
    axisFormat %s
    
    section 版本历史
    Version 1 (原始)    :done, v1, 0, 1
    Version 2 (更新)    :done, v2, 1, 2
    Version 3 (修改)    :done, v3, 2, 3
    Version 4 (恢复到V1) :active, v4, 3, 4
    
    section 当前状态
    当前版本 V4        :crit, current, 3, 4
```

## 使用示例

### 请求示例

```json
{
    "version": 2,
    "commitMessage": "恢复到修复bug前的版本"
}
```

### 响应示例

```json
{
    "metadata": {
        "fileId": "abc123def456",
        "userId": "user123",
        "fileName": "重要文档.pdf",
        "fileType": "document",
        "contentType": "application/pdf",
        "size": 2045678,
        "version": 5,
        "updatedAt": 1634567890,
        "commitMessage": "恢复到修复bug前的版本"
    },
    "message": "File successfully reverted to version 2. New version: 5"
}
```

## 关键特性说明

### 1. 版本完整性

- 保留所有历史版本记录
- 恢复操作不会删除任何现有版本
- 创建新版本来表示恢复状态

### 2. 数据一致性

- 使用数据库事务确保原子性
- 失败时自动回滚所有变更
- 维护版本号的连续性

### 3. 智能恢复

- 检测目标版本是否为当前版本
- 验证目标版本的可用性
- 支持自定义恢复说明

### 4. 审计跟踪

- 记录恢复操作的详细信息
- 保留操作时间和用户信息
- 支持恢复历史查询

## 错误处理场景

### 1. 目标版本不存在

```mermaid
flowchart TD
    A[查询目标版本] --> B{版本存在?}
    B -->|否| C[返回版本不存在错误]
    C --> D[记录错误日志]
    D --> E[HTTP 404响应]
```

### 2. 版本状态异常

```mermaid
flowchart TD
    A[检查版本状态] --> B{状态正常?}
    B -->|否| C[检查具体状态]
    C --> D{已删除?}
    D -->|是| E[返回版本已删除错误]
    D -->|否| F[返回版本不可用错误]
```

### 3. 权限验证失败

```mermaid
flowchart TD
    A[验证用户权限] --> B{用户匹配?}
    B -->|否| C[返回权限不足错误]
    C --> D[记录权限检查失败]
    D --> E[HTTP 403响应]
```

## 性能优化

### 1. 查询优化

- 使用复合索引加速版本查询
- 预加载必要的版本信息
- 避免不必要的数据传输

### 2. 事务优化

- 最小化事务持有时间
- 合理设置事务隔离级别
- 及时释放数据库连接

### 3. 缓存策略

- 缓存热点文件的版本信息
- 使用版本号作为缓存键
- 恢复后更新相关缓存

## 扩展功能

### 1. 批量恢复

```mermaid
flowchart TD
    A[批量恢复请求] --> B[验证所有文件权限]
    B --> C[逐个处理恢复]
    C --> D[收集处理结果]
    D --> E[返回批量恢复响应]
```

### 2. 恢复预览

```mermaid
sequenceDiagram
    participant Client
    participant Logic
    participant Storage
    Client->>Logic: 请求恢复预览
    Logic->>Storage: 获取目标版本内容摘要
    Storage-->>Logic: 返回内容预览
    Logic-->>Client: 展示恢复预览信息
```

### 3. 恢复确认

```mermaid
stateDiagram-v2
    [*] --> PreviewRequested: 请求预览
    PreviewRequested --> PreviewShown: 显示预览
    PreviewShown --> ConfirmRevert: 用户确认
    PreviewShown --> Cancelled: 用户取消
    ConfirmRevert --> RevertCompleted: 恢复完成
    Cancelled --> [*]: 操作取消
    RevertCompleted --> [*]: 操作完成
```

## 监控指标

- 版本恢复操作频率
- 恢复操作成功率
- 平均恢复处理时间
- 最常恢复的版本分析

## 安全考虑

1. **权限控制**：严格验证用户对文件的操作权限
2. **版本验证**：确保恢复的版本数据完整性
3. **操作审计**：记录所有恢复操作的详细日志
4. **并发控制**：防止同时进行的版本操作冲突

整个版本恢复流程设计确保了数据的安全性和一致性，为用户提供了可靠的版本回退功能。
