# KV 存储说明与基准对比

本文档介绍 `pkg/internal/storage/kv` 包的主要接口、使用方式，以及本地基准测试结果对比与注意事项。

## 接口概览

- 包路径：`github.com/yeisme/notevault/pkg/internal/storage/kv`
- 关键类型：

  - `type KVStore interface`：统一的键值存储抽象
    - `Get(ctx, key) ([]byte, error)` 读取
    - `Set(ctx, key, value []byte, ttl time.Duration) error` 写入（可选 TTL）
    - `Delete(ctx, key) error` 删除
    - `Exists(ctx, key) (bool, error)` 是否存在
    - `Keys(ctx, pattern string) ([]string, error)` 键列表（仅调试用途）
    - `Close() error` 关闭
  - `type Client struct { KVStore }`：轻量封装，便于通过配置创建

- 已实现的后端（并通过工厂注册）：
  - Memory（内存）
  - Groupcache（本地缓存 + 可选 peers）
  - Redis（原生 TTL）
  - NATS KV（JetStream）

## 创建与使用

### 方式一：根据全局配置创建（推荐）

```go
ctx := context.Background()
cli, err := kv.New(ctx)
if err != nil {
    // handle error
}

defer cli.Close()

if err := cli.Set(ctx, "foo", []byte("bar"), 10*time.Second); err != nil {
    // handle error
}

val, err := cli.Get(ctx, "foo")
```

- 对应配置结构见 `pkg/configs/kv.go`，支持的类型：`memory`、`redis`、`nats`、`groupcache`。
- 默认值：Redis `localhost:6379`，NATS `localhost:4222`，Groupcache 本地。

### 方式二：显式选择后端

```go
store, err := kv.NewKVStore(ctx, kv.KVTypeRedis, &configs.RedisKVConfig{Addr: "127.0.0.1:6379"})
// or: kv.KVTypeNATS, kv.KVTypeGroupcache, kv.KVTypeMemory
```

## TTL 语义

- Redis：使用原生 TTL。
- NATS / Memory / Groupcache：通过统一的值包装方案实现"每个键的 TTL"，读取路径进行懒过期清理：
  - 存储时将值编码为 `NVTTL1:` 前缀 + JSON 包含 `v`（原值）与 `e`（过期时间戳）。
  - 读取/判断存在/列举时解码并对过期键做懒清理。

该做法为非强一致过期策略，满足大多数业务；如需更强一致性，可使用原生支持 TTL 的后端（如 Redis）或引入后台清理任务。

## 本地测试环境

在 `pkg/internal/storage/kv/docker-compose.yaml` 提供了最小化依赖：

- Redis（6379）
- NATS（4222，8222），已启用 JetStream

启动：

```powershell
cd pkg/internal/storage/kv
docker compose up -d
```

## 基准测试

基准位于 `pkg/internal/storage/kv/kv_test.go`。

- Memory 与 Groupcache 始终参与基准。
- Redis 与 NATS 通过环境变量开启：
  - `ENABLE_REDIS_BENCH=1`（可选 `REDIS_ADDR`）
  - `ENABLE_NATS_BENCH=1`（可选 `NATS_URL`、`NATS_BUCKET`）

示例（PowerShell）：

```powershell
$env:ENABLE_REDIS_BENCH="1"
$env:ENABLE_NATS_BENCH="1"

go test ./pkg/internal/storage/kv -bench=. -benchmem -run=^$
```

### 一组参考结果（你的机器可能不同）

```txt
goos: windows
goarch: amd64
pkg: github.com/yeisme/notevault/pkg/internal/storage/kv
cpu: 11th Gen Intel(R) Core(TM) i5-11260H @ 2.60GHz
BenchmarkMemoryKV/memory/size=32/ttl=0s-12               1757196               681.5 ns/op           200 B/op          8 allocs/op
BenchmarkMemoryKV/memory/size=32/ttl=5s-12                370161              3196 ns/op             745 B/op         18 allocs/op
BenchmarkMemoryKV/memory/size=1024/ttl=0s-12              858921              1935 ns/op            2185 B/op          8 allocs/op
BenchmarkMemoryKV/memory/size=1024/ttl=5s-12               71724             19408 ns/op            6842 B/op         18 allocs/op
BenchmarkMemoryKV/memory/size=65536/ttl=0s-12              19237             73263 ns/op          131313 B/op          8 allocs/op
BenchmarkMemoryKV/memory/size=65536/ttl=5s-12                687           1598501 ns/op          420304 B/op         18 allocs/op
BenchmarkMemoryKV/memory/parallel-12                      332281              3834 ns/op            2196 B/op          8 allocs/op
BenchmarkGroupcacheKV/groupcache/size=32/ttl=0s-12        443770              2477 ns/op             362 B/op         11 allocs/op
BenchmarkGroupcacheKV/groupcache/size=32/ttl=5s-12        475874              4548 ns/op             554 B/op         13 allocs/op
BenchmarkGroupcacheKV/groupcache/size=1024/ttl=0s-12              415680              3233 ns/op            1208 B/op          7 allocs/op
BenchmarkGroupcacheKV/groupcache/size=1024/ttl=5s-12              182647              9305 ns/op            4448 B/op         11 allocs/op
BenchmarkGroupcacheKV/groupcache/size=65536/ttl=0s-12              32446             36460 ns/op           65720 B/op          7 allocs/op
BenchmarkGroupcacheKV/groupcache/size=65536/ttl=5s-12               5822            275092 ns/op          270846 B/op         11 allocs/op
BenchmarkGroupcacheKV/groupcache/parallel-12                      159918              7569 ns/op            4550 B/op         17 allocs/op
BenchmarkRedisKV/redis/size=32/ttl=0s-12                             307           3332640 ns/op             659 B/op         15 allocs/op
BenchmarkRedisKV/redis/size=32/ttl=5s-12                             427           3672490 ns/op             659 B/op         15 allocs/op
BenchmarkRedisKV/redis/size=1024/ttl=0s-12                           422           3325097 ns/op            2756 B/op         15 allocs/op
BenchmarkRedisKV/redis/size=1024/ttl=5s-12                           381           3197984 ns/op            2755 B/op         15 allocs/op
BenchmarkRedisKV/redis/size=65536/ttl=0s-12                          216           4760631 ns/op          139841 B/op         15 allocs/op
BenchmarkRedisKV/redis/size=65536/ttl=5s-12                          196           7542791 ns/op          139844 B/op         15 allocs/op
BenchmarkRedisKV/redis/parallel-12                                  1591            753017 ns/op            2769 B/op         16 allocs/op
BenchmarkNATSKV/nats/size=32/ttl=0s-12                               274           5137145 ns/op            4396 B/op         76 allocs/op
BenchmarkNATSKV/nats/size=32/ttl=5s-12                               200           5941724 ns/op            4992 B/op         86 allocs/op
BenchmarkNATSKV/nats/size=1024/ttl=0s-12                             181           6374707 ns/op            7077 B/op         76 allocs/op
BenchmarkNATSKV/nats/size=1024/ttl=5s-12                             242           6592215 ns/op           11875 B/op         86 allocs/op
BenchmarkNATSKV/nats/size=65536/ttl=0s-12                            100          16074110 ns/op          168041 B/op         76 allocs/op
BenchmarkNATSKV/nats/size=65536/ttl=5s-12                            112          15083775 ns/op          471397 B/op         86 allocs/op
BenchmarkNATSKV/nats/parallel-12                                    1237            876841 ns/op            7077 B/op         77 allocs/op
PASS
coverage: 45.9% of statements
ok      github.com/yeisme/notevault/pkg/internal/storage/kv     62.281s
```

### 基准结果解读与建议

- 总体趋势：

  - MemoryKV 最快，适合单进程/测试场景；Groupcache 次之，更适合"读多的热点缓存"。
  - Redis/NATS 明显慢于内存类后端，属于跨进程/网络访问的典型代价；在 Windows + Docker 下延迟更高属于常态。
  - 相比 Redis，NATS KV 更慢一些，适合轻量元数据或需要复用 NATS/JetStream 基础设施的场景。

- 负载维度：

  - TTL=0 与 TTL=5s 差距明显：TTL>0 会触发值包装/JSON 编解码与校验，带来额外分配和耗时。
  - size 从 32 → 1024 → 65536 增大时，耗时与内存占用随之上升，复制/序列化成本主导趋势；跨进程后端还叠加网络传输开销。
  - 并行用例（parallel）下，Memory/Groupcache 可很好扩展；Redis/NATS 受网络与服务端吞吐影响，提升有限。

- 选型建议：

  - 简单/单实例/开发测试：优先 MemoryKV。
  - 多实例、需要本地热缓存：Groupcache；若需要跨进程共享和持久化：Redis。
  - 强 TTL 语义、丰富运维与生态：Redis 更优（原生 TTL、可观测性好）。
  - 已有 NATS/JetStream 基础并希望轻量存元信息：NATS KV 可用，但要接受其性能特征。

## 常见问题与注意事项

- NATS KV 键名限制：不允许包含冒号等特殊字符。本项目基准已将键名统一改用连字符（例如：`bench-nats-1`）。
- Keys(pattern) 仅用于调试，不建议在生产高频使用；不同后端的匹配能力也各不相同。
- 若需强 TTL 语义与丰富查询，优先选用 Redis；如果只需要本地缓存/进程内存，选用 Memory 或 Groupcache。

## 下一步

- 如需跨进程分布式缓存，可结合 Groupcache peers 或直接使用 Redis。
- 可增加后台清理任务以配合 Memory/Groupcache 的懒过期策略（可选）。
- 若需要进一步的功能（如批量操作、前缀扫描），可在接口层扩展并按后端能力降级实现。
