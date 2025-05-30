# 文件删除处理流程详解

本文档详细介绍文件删除的处理流程，并通过多个mermaid图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始删除] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D[查询文件元数据]
    D --> E{文件存在?}
    E -->|否| F[返回文件不存在错误]
    E -->|是| G{用户有权限?}
    G -->|否| H[返回权限错误]
    G -->|是| I[开始事务处理]
    I --> J[软删除文件记录]
    J --> K[软删除版本记录]
    K --> L[删除标签关联]
    L --> M{事务成功?}
    M -->|否| N[回滚事务]
    M -->|是| O[返回成功响应]
    N --> P[返回删除失败错误]
```

## 详细步骤分析

### 1. 请求处理与身份验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Context
    
    Client->>Handler: 发送删除请求(FileID)
    Handler->>Handler: 解析请求参数
    Handler->>Logic: 创建删除逻辑并传递请求
    Logic->>Context: 从上下文获取userId
    alt 获取成功
        Context-->>Logic: 返回userId
        Logic->>Logic: 继续处理流程
    else 获取失败
        Context-->>Logic: 返回空或错误
        Logic->>Logic: 使用默认用户ID(测试模式)
    end
```

### 2. 文件查询与权限验证

```mermaid
flowchart TD
    A[初始化数据库查询] --> B[根据FileID查询文件]
    B --> C[添加DeletedAt=0条件]
    C --> D{查询成功?}
    D -->|否| E{ErrRecordNotFound?}
    E -->|是| F[返回文件不存在错误]
    E -->|否| G[返回数据库查询错误]
    D -->|是| H{检查文件所有权}
    H -->|UserID不匹配| I[返回权限错误]
    H -->|UserID匹配| J[继续删除流程]
```

### 3. 事务处理流程

```mermaid
sequenceDiagram
    participant Logic
    participant Transaction
    participant FileTable
    participant VersionTable
    participant TagTable
    
    Logic->>Transaction: 开始数据库事务
    Transaction->>FileTable: 软删除文件记录
    alt 文件删除成功
        FileTable-->>Transaction: 返回成功
        Transaction->>VersionTable: 软删除所有版本
        alt 版本删除成功
            VersionTable-->>Transaction: 返回成功
            Transaction->>TagTable: 删除标签关联
            alt 标签关联删除成功
                TagTable-->>Transaction: 返回成功
                Transaction-->>Logic: 提交事务
            else 标签关联删除失败
                TagTable-->>Transaction: 返回错误(记录日志)
                Transaction-->>Logic: 提交事务(非关键错误)
            end
        else 版本删除失败
            VersionTable-->>Transaction: 返回错误(记录日志)
            Transaction-->>Logic: 提交事务(非关键错误)
        end
    else 文件删除失败
        FileTable-->>Transaction: 返回错误
        Transaction-->>Logic: 回滚事务
        Logic-->>Client: 返回删除失败错误
    end
```

### 4. 软删除机制详解

```mermaid
flowchart LR
    A[文件记录] --> B[设置deleted_at时间戳]
    A --> C[更新status为3-待删除]
    A --> D[更新updated_at时间戳]
    E[版本记录] --> F[设置deleted_at时间戳]
    E --> G[更新status为2-已替换]
    H[标签关联] --> I[物理删除关联记录]
    
    subgraph 软删除策略
        B
        C
        D
        F
        G
    end
    
    subgraph 物理删除策略
        I
    end
```

## 数据库操作分析

### 文件状态变更

```mermaid
stateDiagram-v2
    [*] --> Normal : 创建文件
    Normal --> PendingDeletion : 用户删除
    Normal --> Archived : 归档
    Archived --> PendingDeletion : 删除归档文件
    PendingDeletion --> AdminTrash : 管理员操作
    AdminTrash --> [*] : 永久删除
    
    note right of Normal : status=0
    note right of Archived : status=1
    note right of AdminTrash : status=2
    note right of PendingDeletion : status=3
```

### 版本状态变更

```mermaid
stateDiagram-v2
    [*] --> Active : 创建版本
    Active --> Outdated : 新版本覆盖
    Active --> Replaced : 删除操作
    Outdated --> Replaced : 删除操作
    Replaced --> [*] : 清理操作
    
    note right of Active : status=0
    note right of Outdated : status=1
    note right of Replaced : status=2
```

## 数据库模型关系及删除影响

```mermaid
erDiagram
    FILE ||--o{ FILE_VERSION : has
    FILE ||--o{ FILE_TAG : has
    TAG ||--o{ FILE_TAG : has
    
    FILE {
        string FileID PK
        string UserID
        string FileName
        string FileType
        string ContentType
        int64 Size
        string Path
        int64 CreatedAt
        int64 UpdatedAt
        int64 DeletedAt "软删除标记"
        int16 Status "状态码"
        int64 TrashedAt
        int32 CurrentVersion
        string Description
    }
    
    FILE_VERSION {
        string VersionID PK
        string FileID FK
        int32 VersionNumber
        int64 Size
        string Path
        string ContentType
        int64 CreatedAt
        int64 DeletedAt "软删除标记"
        int16 Status "状态码"
        string CommitMessage
    }
    
    TAG {
        string TagID PK
        string Name
    }
    
    FILE_TAG {
        string FileID FK "物理删除"
        string TagID FK
    }
```

## 错误处理流程

```mermaid
flowchart TD
    A[处理过程发生错误] --> B{错误类型?}
    B -->|文件不存在| C[返回404错误<br/>File not found or already deleted]
    B -->|权限不足| D[返回403错误<br/>No permission to delete this file]
    B -->|数据库查询错误| E[记录错误日志<br/>返回500错误]
    B -->|事务执行错误| F[记录错误日志<br/>回滚事务<br/>返回500错误]
    B -->|版本删除失败| G[记录警告日志<br/>继续主流程]
    B -->|标签关联删除失败| H[记录警告日志<br/>继续主流程]
    
    C --> I[返回给客户端]
    D --> I
    E --> I
    F --> I
    G --> J[继续事务]
    H --> J
    J --> K[完成删除操作]
```

## 安全性考虑

```mermaid
flowchart TD
    A[接收删除请求] --> B[验证用户身份]
    B --> C[检查文件所有权]
    C --> D[验证文件状态]
    D --> E{文件已删除?}
    E -->|是| F[拒绝操作]
    E -->|否| G[执行删除]
    
    subgraph 安全检查
        B
        C
        D
        E
    end
```

## OSS文件保留策略

```mermaid
flowchart LR
    A[用户删除文件] --> B[数据库软删除]
    B --> C[OSS文件保留]
    C --> D[管理员审核期]
    D --> E{需要恢复?}
    E -->|是| F[恢复文件记录]
    E -->|否| G[定期清理任务]
    G --> H[永久删除OSS文件]
    
    subgraph 保留期间
        C
        D
    end
    
    subgraph 恢复机制
        F
    end
    
    subgraph 清理机制
        G
        H
    end
```

## 成功删除响应流程

```mermaid
sequenceDiagram
    participant Logic
    participant Database
    participant Client
    
    Logic->>Database: 执行软删除事务
    Database-->>Logic: 返回成功
    Logic->>Logic: 构建响应消息
    Logic->>Logic: 记录成功日志
    Logic-->>Client: 返回删除成功响应
    
    Note over Client: 响应包含:<br/>- 成功消息<br/>- 用户ID<br/>- 文件ID<br/>- 文件名<br/>- OSS保留说明
```

## 日志记录策略

```mermaid
flowchart TD
    A[删除操作开始] --> B[记录操作请求]
    B --> C{操作成功?}
    C -->|是| D[记录成功日志<br/>Info级别]
    C -->|否| E[记录错误日志<br/>Error级别]
    
    D --> F[包含信息:<br/>- 用户ID<br/>- 文件ID<br/>- 文件名<br/>- 删除时间]
    
    E --> G[包含信息:<br/>- 错误类型<br/>- 错误详情<br/>- 用户ID<br/>- 文件ID]
```

## 关键设计特点

### 1. 软删除机制

- **文件记录**: 设置 `deleted_at` 时间戳，状态更新为3（待删除）
- **版本记录**: 设置 `deleted_at` 时间戳，状态更新为2（已替换）
- **OSS文件**: 保留不删除，便于数据恢复和审计

### 2. 事务一致性

- 使用数据库事务确保所有相关记录的一致性更新
- 核心操作失败时自动回滚
- 非关键操作失败时记录日志但不影响主流程

### 3. 权限控制

- 验证文件所有权，防止越权删除
- 支持软删除状态检查，避免重复删除

### 4. 错误处理

- 详细的错误分类和处理
- 适当的日志记录级别
- 清晰的错误消息返回

### 5. 数据保护

- OSS文件保留策略，支持数据恢复
- 软删除机制，避免意外数据丢失
- 详细的操作日志，支持审计追踪

整个删除流程设计注重数据安全和操作可逆性，通过软删除机制和OSS文件保留，为用户提供数据保护的同时，也为系统管理员提供了灵活的数据管理能力。
