# 获取文件版本差异处理流程详解

本文档详细介绍获取文件版本差异的处理流程，并通过多个 mermaid 图表进行可视化说明。

## 整体流程概览

```mermaid
flowchart TD
    A[开始获取版本差异] --> B[解析请求参数]
    B --> C[验证用户身份]
    C --> D[查询文件基本信息]
    D --> E{文件存在?}
    E -->|否| F[返回文件不存在错误]
    E -->|是| G{用户有权限?}
    G -->|否| H[返回权限错误]
    G -->|是| I[验证版本号参数]
    I --> J{版本号有效?}
    J -->|否| K[返回版本号无效错误]
    J -->|是| L[查询基准版本]
    L --> M[查询目标版本]
    M --> N{两个版本都存在?}
    N -->|否| O[返回版本不存在错误]
    N -->|是| P[检查文件类型]
    P --> Q{支持差异比较?}
    Q -->|否| R[返回不支持差异比较]
    Q -->|是| S[获取版本内容]
    S --> T[计算文件差异]
    T --> U[格式化差异结果]
    U --> V[返回差异响应]
```

## 详细步骤分析

### 1. 请求处理与验证

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Logic
    participant Database
    Client->>Handler: GET /api/v1/files/:fileId/versions/diff
    Note over Client: baseVersion=1&targetVersion=3
    Handler->>Logic: GetFileVersionDiff(req)
    Logic->>Database: 查询文件基本信息
    Database-->>Logic: 返回文件元数据
    Logic->>Logic: 验证用户权限
    Logic->>Database: 查询基准版本信息
    Database-->>Logic: 返回基准版本详情
    Logic->>Database: 查询目标版本信息
    Database-->>Logic: 返回目标版本详情
    Logic-->>Handler: 开始差异计算
```

### 2. 版本内容获取流程

```mermaid
flowchart TD
    A[获取版本内容] --> B[查询基准版本路径]
    B --> C[从存储获取基准内容]
    C --> D[查询目标版本路径]
    D --> E[从存储获取目标内容]
    E --> F{内容获取成功?}
    F -->|否| G[返回内容获取失败]
    F -->|是| H[验证内容完整性]
    H --> I[准备差异计算]
```

### 3. 文件类型处理

```mermaid
flowchart TD
    A[检查文件类型] --> B{文本文件?}
    B -->|是| C[使用文本差异算法]
    B -->|否| D{图片文件?}
    D -->|是| E[计算图片差异]
    D -->|否| F{代码文件?}
    F -->|是| G[使用代码差异算法]
    F -->|否| H[使用二进制差异]
    C --> I[生成差异结果]
    E --> I
    G --> I
    H --> I
```

### 4. 差异计算算法

```mermaid
sequenceDiagram
    participant Logic
    participant DiffEngine
    participant Storage
    participant Formatter
    Logic->>Storage: 获取基准版本内容
    Storage-->>Logic: 返回基准内容
    Logic->>Storage: 获取目标版本内容
    Storage-->>Logic: 返回目标内容
    Logic->>DiffEngine: 计算内容差异
    DiffEngine->>DiffEngine: 执行差异算法
    Note over DiffEngine: 使用Myers算法或类似
    DiffEngine-->>Logic: 返回差异数据
    Logic->>Formatter: 格式化差异结果
    Formatter-->>Logic: 返回格式化的差异
    Logic-->>Logic: 构建响应数据
```

## 差异算法选择

```mermaid
flowchart TD
    A[文件类型判断] --> B{文本类型?}
    B -->|纯文本| C[逐行差异算法]
    B -->|JSON| D[结构化差异算法]
    B -->|XML/HTML| E[DOM差异算法]
    B -->|代码文件| F[语法感知差异]
    B -->|Markdown| G[标记语言差异]
    B -->|二进制| H[字节级差异]

    C --> I[Myers算法]
    D --> J[JSON Patch算法]
    E --> K[XML Diff算法]
    F --> L[AST差异算法]
    G --> M[Markdown差异算法]
    H --> N[二进制差异算法]
```

## 差异结果格式

```mermaid
classDiagram
    class FileVersionDiffRequest {
        +string FileID
        +int BaseVersion
        +int TargetVersion
    }

    class FileVersionDiffResponse {
        +string FileID
        +int BaseVersion
        +int TargetVersion
        +string DiffContent
        +string Message
        +DiffStats Stats
    }

    class DiffStats {
        +int LinesAdded
        +int LinesDeleted
        +int LinesModified
        +int TotalChanges
    }

    FileVersionDiffRequest --> FileVersionDiffResponse: 处理后返回
    FileVersionDiffResponse --> DiffStats: 包含差异统计
```

## 差异输出格式示例

### 1. 统一差异格式 (Unified Diff)

```mermaid
flowchart TD
    A[原始文件内容] --> B[目标文件内容]
    B --> C[生成差异块]
    C --> D[标记行号信息]
    D --> E[添加上下文行]
    E --> F[格式化输出]

    F --> G["@@ -1,4 +1,6 @@
    line 1
    -line 2 (deleted)
    +line 2 (modified)
    +line 2.5 (added)
    line 3
    line 4"]
```

### 2. 结构化差异格式

```json
{
  "changes": [
    {
      "type": "delete",
      "lineNumber": 2,
      "content": "old content"
    },
    {
      "type": "add",
      "lineNumber": 2,
      "content": "new content"
    }
  ]
}
```

## 使用示例

### 请求示例

```http
GET /api/v1/files/abc123def456/versions/diff?baseVersion=1&targetVersion=3 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <jwt-token>
```

### 响应示例

```json
{
  "fileId": "abc123def456",
  "baseVersion": 1,
  "targetVersion": 3,
  "diffContent": "@@ -1,10 +1,12 @@\n # 项目文档\n \n-## 旧的章节标题\n+## 新的章节标题\n \n 这是一些内容...\n \n+## 新增的章节\n+这是新增的内容。\n+\n ## 结论\n 项目总结内容。",
  "stats": {
    "linesAdded": 3,
    "linesDeleted": 1,
    "linesModified": 1,
    "totalChanges": 5
  },
  "message": "Successfully generated diff between version 1 and version 3"
}
```

## 关键特性说明

### 1. 多格式支持

- 支持多种文件类型的差异比较
- 智能选择最适合的差异算法
- 可扩展的差异处理器架构

### 2. 性能优化

- 大文件分块处理
- 缓存常用版本内容
- 异步处理复杂差异计算

### 3. 用户友好

- 清晰的差异可视化
- 详细的统计信息
- 支持多种输出格式

### 4. 安全性

- 严格的权限验证
- 内容访问控制
- 操作日志记录

## 错误处理场景

### 1. 版本不存在

```mermaid
flowchart TD
    A[查询版本] --> B{基准版本存在?}
    B -->|否| C[返回基准版本不存在]
    B -->|是| D{目标版本存在?}
    D -->|否| E[返回目标版本不存在]
    D -->|是| F[继续处理]
```

### 2. 文件类型不支持

```mermaid
flowchart TD
    A[检查文件类型] --> B{支持差异比较?}
    B -->|否| C[返回不支持错误]
    C --> D[建议下载版本比较]
    B -->|是| E[执行差异计算]
```

### 3. 内容获取失败

```mermaid
flowchart TD
    A[获取文件内容] --> B{内容完整?}
    B -->|否| C[返回内容损坏错误]
    C --> D[记录错误详情]
    B -->|是| E[继续差异计算]
```

## 性能考虑

### 1. 缓存策略

```mermaid
flowchart TD
    A[差异请求] --> B{缓存中存在?}
    B -->|是| C[返回缓存结果]
    B -->|否| D[计算差异]
    D --> E[存储到缓存]
    E --> F[返回差异结果]
```

### 2. 大文件处理

```mermaid
sequenceDiagram
    participant Client
    participant Logic
    participant Worker
    participant Storage
    Client->>Logic: 大文件差异请求
    Logic->>Worker: 异步处理任务
    Worker->>Storage: 分块读取文件内容
    Storage-->>Worker: 返回文件块
    Worker->>Worker: 分块计算差异
    Worker-->>Logic: 返回差异结果
    Logic-->>Client: 返回差异响应
```
