# Store 层设计

这个目录用于持久化与数据访问，业务层只依赖接口，不直接依赖存储实现。

## 设计原则

- **TaskManager 内存**：SubTasks、实时运行状态（短期、不持久化）
- **Store 持久化**：任务元信息、最终结果路径（长期、可恢复）

## TODO：定义存储接口

新建 `internal/store/interface.go`：

```go
type Store interface {
    GetTask(ctx context.Context, id string) (*TaskRecord, error)
    SaveTask(ctx context.Context, task *TaskRecord) error
    UpdateTaskStatus(ctx context.Context, id string, status string) error
    DeleteTask(ctx context.Context, id string) error
}
```

## TODO：定义存储模型

新建 `internal/store/models.go`：

```go
type TaskRecord struct {
    ID         string    `json:"id"`
    Status     string    `json:"status"`      // pending, processing, completed, failed
    PDFPath    string    `json:"pdf_path"`    // 原始 PDF 路径
    ResultPath string    `json:"result_path"` // 结果 Markdown 路径
    TotalPages int       `json:"total_pages"`
    Error      string    `json:"error,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

## TODO：接入 TaskManager

- `internal/task/manager.go` 接受 `store.Store` 接口实例
- 任务创建/状态变更时同步写入 Store
- 服务启动时从 Store 恢复已完成任务的索引（可选，后续优化）
