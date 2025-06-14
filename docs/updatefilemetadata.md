# 更新文件元数据处理流程详解

本文档详细介绍更新文件元数据的处理流程，并通过多个mermaid图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始更新元数据] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D[查询原文件信息]
    D --> E{文件存在?}
    E -->|否| F[返回文件不存在错误]
    E -->|是| G{用户有权限?}
    G -->|否| H[返回权限错误]
    G -->|是| I[检查更新内容]
    I --> J{有实际更新?}
    J -->|否| K[返回无更新消息]
    J -->|是| L[开始数据库事务]
    L --> M[更新文件元数据]
    M --> N[创建新版本记录]
    N --> O[处理标签更新]
    O --> P{事务成功?}
    P -->|否| Q[回滚事务]
    P -->|是| R[返回更新后的元数据]
    Q --> S[返回更新失败错误]
```

## 详细步骤分析

### 1. 请求处理与验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Database
    Client->>Handler: PUT /api/v1/files/metadata/:fileId
    Handler->>Logic: UpdateFileMetadata(req)
    Logic->>Database: 查询原文件信息
    Database-->>Logic: 返回文件元数据
    Logic->>Logic: 验证用户权限
    Logic->>Logic: 检查更新内容
    Logic-->>Handler: 返回处理结果
```

### 2. 元数据更新处理

```mermaid
flowchart TD
    A[检查更新字段] --> B{文件名有更新?}
    B -->|是| C[验证文件名有效性]
    B -->|否| D{描述有更新?}
    C --> E{文件名有效?}
    E -->|否| F[返回文件名无效错误]
    E -->|是| D
    D -->|是| G[更新描述字段]
    D -->|否| H{标签有更新?}
    G --> H
    H -->|是| I[处理标签变更]
    H -->|否| J[检查提交消息]
    I --> J
    J --> K[准备更新数据]
```

### 3. 版本管理流程

```mermaid
sequenceDiagram
    participant Logic
    participant Database
    participant Transaction
    Logic->>Transaction: 开始事务
    Transaction->>Database: 更新文件基本信息
    Database-->>Transaction: 更新成功
    Transaction->>Database: 递增版本号
    Database-->>Transaction: 版本号更新成功
    Transaction->>Database: 创建新版本记录
    Note over Database: 记录元数据变更历史
    Database-->>Transaction: 版本记录创建成功
    Transaction->>Database: 更新文件updated_at
    Database-->>Transaction: 时间戳更新成功
    Transaction-->>Logic: 提交事务
```

### 4. 标签管理详细流程

```mermaid
flowchart TD
    A[开始标签处理] --> B[获取当前标签]
    B --> C[解析新标签列表]
    C --> D[计算标签差异]
    D --> E{有需要删除的标签?}
    E -->|是| F[删除旧标签关联]
    E -->|否| G{有需要添加的标签?}
    F --> G
    G -->|是| H[处理新标签]
    G -->|否| I[标签处理完成]
    H --> J[检查标签是否存在]
    J --> K{标签存在?}
    K -->|否| L[创建新标签]
    K -->|是| M[创建文件标签关联]
    L --> M
    M --> N{还有新标签?}
    N -->|是| H
    N -->|否| I
```

## 数据库变更示意图

```mermaid
erDiagram
    FILE ||--o{ FILE_VERSION : creates
    FILE ||--o{ FILE_TAG : updates
    TAG ||--o{ FILE_TAG : references
    
    FILE {
        string file_name "可更新"
        string description "可更新"
        int64 updated_at "自动更新"
        int current_version "自动递增"
    }
    
    FILE_VERSION {
        string version_id "新增记录"
        int version_number "递增"
        string commit_message "元数据更新记录"
        int64 created_at "创建时间"
    }
    
    FILE_TAG {
        string file_id "保持不变"
        string tag_id "可能变更"
    }
```

## 版本历史记录

```mermaid
stateDiagram-v2
    [*] --> Version1: 文件创建
    Version1 --> Version2: 元数据更新
    Version2 --> Version3: 再次更新
    Version3 --> Version4: 标签修改
    
    state Version1 {
        [*] --> Created
        Created: 原始文件信息
    }
    
    state Version2 {
        [*] --> MetadataUpdated
        MetadataUpdated: 文件名或描述更新
    }
    
    state Version3 {
        [*] --> MetadataUpdated2
        MetadataUpdated2: 进一步元数据更新
    }
    
    state Version4 {
        [*] --> TagsModified
        TagsModified: 标签关联更新
    }
```

## 响应结构说明

```mermaid
classDiagram
    class UpdateFileMetadataRequest {
        +string FileID
        +string FileName
        +string Description
        +[]string Tags
        +string CommitMessage
    }
    
    class UpdateFileMetadataResponse {
        +FileMetadata Metadata
        +string Message
    }
    
    class FileMetadata {
        +string FileID
        +string FileName
        +string Description
        +[]string Tags
        +int Version
        +int64 UpdatedAt
        +string CommitMessage
    }
    
    UpdateFileMetadataRequest --> UpdateFileMetadataResponse: 处理后返回
    UpdateFileMetadataResponse --> FileMetadata: 包含更新后的元数据
```

## 关键特性说明

### 1. 版本控制
- 每次元数据更新都会创建新版本记录
- 版本号自动递增
- 保留完整的变更历史

### 2. 标签管理
- 支持标签的增加、删除和修改
- 自动创建不存在的标签
- 清理不再使用的标签关联

### 3. 原子性操作
- 所有更新操作在单个事务中完成
- 失败时自动回滚所有变更
- 确保数据一致性

### 4. 智能更新检测
- 只有实际发生变化的字段才会更新
- 避免不必要的版本创建
- 优化数据库性能

## 使用示例

### 请求示例

```json
{
    "fileName": "新的文件名.pdf",
    "description": "更新后的文件描述",
    "tags": ["重要", "工作", "2024"],
    "commitMessage": "更新文件名和描述，添加新标签"
}
```

### 响应示例

```json
{
    "metadata": {
        "fileId": "abc123def456",
        "userId": "user123",
        "fileName": "新的文件名.pdf",
        "description": "更新后的文件描述",
        "tags": ["重要", "工作", "2024"],
        "version": 3,
        "updatedAt": 1634567890,
        "commitMessage": "更新文件名和描述，添加新标签"
    },
    "message": "File metadata updated successfully to version 3"
}
```

## 错误处理场景

### 1. 文件不存在

```mermaid
flowchart TD
    A[查询文件] --> B{文件存在?}
    B -->|否| C[返回404错误]
    C --> D[记录错误日志]
    D --> E[返回错误响应]
```

### 2. 权限不足

```mermaid
flowchart TD
    A[检查文件所有权] --> B{用户匹配?}
    B -->|否| C[返回403权限错误]
    C --> D[记录权限检查失败]
    D --> E[返回权限错误响应]
```

### 3. 数据库事务失败

```mermaid
flowchart TD
    A[执行数据库更新] --> B{事务成功?}
    B -->|否| C[自动回滚事务]
    C --> D[记录事务失败日志]
    D --> E[返回数据库错误]
```

## 性能优化考虑

1. **差异检测**：只更新实际变化的字段
2. **批量标签处理**：一次性处理所有标签变更
3. **索引优化**：在文件ID和更新时间上建立索引
4. **事务粒度**：使用最小必要的事务范围

## 扩展功能

### 1. 元数据验证
- 文件名长度和字符限制
- 描述内容长度限制
- 标签数量和格式验证

### 2. 变更通知
- 可集成消息队列通知其他服务
- 支持webhook回调
- 审计日志记录

### 3. 批量更新
- 未来可扩展支持批量元数据更新
- 异步处理大量文件
- 进度跟踪和状态报告

整个元数据更新流程设计确保了数据的完整性和一致性，同时提供了灵活的版本管理和标签系统。
