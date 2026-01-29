# Redis 存储实现

该目录实现 `store.Store` 接口的 Redis 版本。

## 设计决策

- **数据结构**：JSON Blob（整条记录序列化存储）
- **Key 格式**：`task:{id}` → JSON string
- **理由**：任务信息简单，状态单向推进，JSON 更直观易调试

## TODO：Redis 客户端初始化

新建 `client.go`：

- 初始化 `redis.Client`
- 从环境变量读取配置：`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`
- 提供 `NewClient()` 构造函数

## TODO：实现 Store 接口

新建 `store.go`：

```go
type RedisStore struct {
    client *redis.Client
    ttl    time.Duration  // 任务过期时间，如 7 天
}

func (s *RedisStore) GetTask(ctx context.Context, id string) (*TaskRecord, error)
func (s *RedisStore) SaveTask(ctx context.Context, task *TaskRecord) error
func (s *RedisStore) UpdateTaskStatus(ctx context.Context, id string, status string) error
func (s *RedisStore) DeleteTask(ctx context.Context, id string) error
```

实现要点：
- `SaveTask`：`SET task:{id} {json} EX {ttl}`
- `GetTask`：`GET task:{id}` → JSON 反序列化
- `UpdateTaskStatus`：读取 → 修改 status/updated_at → 写回
- `DeleteTask`：`DEL task:{id}`

## TODO（后续优化）

- [ ] 过期时自动清理 `output/{task_id}/` 目录（Keyspace Notification 或定时扫描）
- [ ] 启动时恢复任务索引
- [ ] 连接池配置优化
