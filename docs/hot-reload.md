# 热加载配置详解

> 该文档适用于 NoteVault 应用程序的开发和运维人员，帮助他们理解和使用热加载配置功能。

## 概述

NoteVault 应用程序支持配置文件的热加载功能，允许在运行时动态更新配置而无需重启服务器。这对于开发环境和生产环境的配置调整非常有用。本文档详细介绍热加载配置的原理、机制和工作流程。

## 原理

热加载配置基于以下核心组件：

1. **Viper 配置管理库**：用于读取、解析和监听配置文件变化
2. **fsnotify 文件监听器**：监控配置文件的变化事件
3. **配置验证**：确保重新加载的配置符合业务规则
4. **回调机制**：在配置变化时触发应用重启或重新初始化

## 机制

### 配置初始化

在应用程序启动时，通过 `configs.InitConfig()` 函数初始化配置：

```go
func InitConfig(path string) error {
    appViper = viper.New()
    // 设置默认值
    setAllDefaults(appViper)

    // 配置 Viper 读取配置
    // ...

    // 启用热重载
    reloadConfigs(appViper, globalConfig.Server.ReloadConfig, nil)

    return nil
}
```

### 热重载启用

`reloadConfigs` 函数负责设置热重载机制：

```go
func reloadConfigs(v *viper.Viper, isHotReload bool, onReload func()) {
    if !isHotReload {
        return
    }
    onReloadCallback = onReload
    // 启用配置热重载
    v.OnConfigChange(func(e fsnotify.Event) {
        fmt.Println("Config file changed:", e.Name)
        fmt.Println("Reloading configuration...")

        if err := v.Unmarshal(&globalConfig); err != nil {
            fmt.Printf("Error reloading config:\n %v\n", err)
            return
        }

        // 验证重新加载的配置
        if err := rule.ValidateStruct(&globalConfig); err != nil {
            fmt.Printf("Error validating reloaded config:\n %v\n", err)
            return
        }

        fmt.Println("Configuration reloaded successfully")

        // 调用重新加载回调函数
        if onReloadCallback != nil {
            onReloadCallback()
        }
    })
    v.WatchConfig()
}
```

### 应用重启机制

在命令行工具中，设置配置重新加载回调来触发应用重启：

```go
// 设置配置重新加载回调
configs.SetReloadCallback(func() {
    select {
    case restartChan <- struct{}{}:
    default {
        // 如果通道已满，忽略这次重新启动信号
    }
})
```

## 工作流程

### 1. 应用程序启动

1. 执行 `cmd.Execute()` 启动命令行工具
2. 在 `PersistentPreRun` 中调用 `configs.InitConfig()` 初始化配置
3. 如果启用热重载，调用 `reloadConfigs()` 设置文件监听

### 2. 配置监听

1. Viper 通过 `v.WatchConfig()` 启动文件监听
2. fsnotify 监听配置文件的变化事件（创建、修改、删除等）

### 3. 配置变化检测

当配置文件发生变化时：

1. fsnotify 触发 `OnConfigChange` 回调
2. Viper 重新读取配置文件
3. 将配置 Unmarshal 到全局配置结构体
4. 使用 `rule.ValidateStruct()` 验证配置有效性

### 4. 应用重启

1. 如果配置验证成功，调用 `onReloadCallback()`
2. 在 `cmd.go` 中，回调函数向 `restartChan` 发送重启信号
3. 主循环检测到重启信号后：
   - 调用 `applications.Shutdown()` 优雅关闭当前应用
   - 继续循环，重新创建新的应用实例
   - 调用 `app.NewApp()` 初始化新应用
   - 启动新的服务器实例

### 5. 服务器重启流程

在 `app.go` 中：

1. `NewApp()` 创建新的应用实例
2. 重新初始化所有组件（追踪、监控、存储等）
3. 创建新的 HTTP 服务器
4. `Run()` 方法启动服务器

## 配置要求

### 启用热重载

在配置文件中设置：

```yaml
server:
  reload_config: true
```

### 支持的配置格式

- YAML (.yaml, .yml)
- JSON (.json)
- TOML (.toml)
- 环境变量 (dotenv)

### 配置验证

所有重新加载的配置都会通过 `rule.ValidateStruct()` 进行验证，确保：

- 必填字段存在
- 数据类型正确
- 值在有效范围内

## 注意事项

1. **配置验证失败**：如果新配置验证失败，会记录错误但保持原有配置不变
2. **并发安全**：配置重载在单 goroutine 中进行，避免并发问题
3. **重启延迟**：配置变化到应用重启之间可能有短暂延迟
4. **资源清理**：重启时会正确关闭旧的服务器和资源连接
5. **信号处理**：支持优雅关闭，避免数据丢失

## 故障排除

### 配置无法重载

- 检查配置文件权限
- 确认 `reload_config` 设置为 `true`
- 查看日志中的错误信息

### 应用无法重启

- 检查端口是否被占用
- 确认存储连接正常
- 查看应用日志

### 性能影响

- 文件监听对性能影响很小
- 重启过程会短暂中断服务
- 建议在低峰期进行配置更新

## 相关代码文件

- `pkg/configs/config.go`：配置管理核心逻辑
- `pkg/cmd/cmd.go`：命令行工具和重启机制
- `pkg/app/app.go`：应用初始化和服务器管理
