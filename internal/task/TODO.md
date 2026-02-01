# TaskManager 重构：接入 Redis Store

## 目标

将 TaskManager 从纯内存存储改为 Redis + 内存混合模式：
- **Redis**：持久化任务元数据（Status、路径、时间戳）
- **内存**：仅保留运行时状态（SubTasks、完成计数、聚合逻辑）

## 现有结构

```
TaskManager
├── tasks map[string]*ParentTask  ← 所有数据都在这里
└── pool *WorkerPool

ParentTask（混合了持久化数据和运行时状态）
├── ID, OriginalPDF, WorkDir, OutputPath  ← 应该持久化
├── Status                                 ← 应该持久化
├── SubTasks map[string]*SubTaskMeta      ← 运行时
├── CompletedCount, FailedTasks           ← 运行时
└── mu, aggregateOnce                     ← 运行时
```

## 目标结构

```
TaskManager
├── store store.Store                    ← Redis（持久化）
├── activeJobs map[string]*JobRuntime    ← 内存（仅运行中任务）
└── pool *WorkerPool

store.TaskRecord（Redis 中）
├── ID, Status
├── PDFPath, ResultPath, WorkDir
├── TotalPages, Error
└── CreatedAt, UpdatedAt

JobRuntime（内存中，任务完成后删除）
├── SubTasks map[string]*SubTaskMeta
├── TotalShards, CompletedCount, FailedCount
├── mu, aggregateOnce
└── (可选) Cancel context.CancelFunc
```

---

## Phase 1：基础设施

### 1.1 完善 Store 层

- [x] `internal/store/models.go` - 定义 TaskRecord
- [x] `internal/store/interface.go` - 定义 Store 接口
- [x] `internal/store/errors.go` - 定义 ErrNotFound
- [x] `internal/store/redis/store.go` - 实现 RedisStore
- [x] `internal/store/redis/store.go` - 添加 NewRedisStore() 构造函数

### 1.2 Redis 客户端初始化

- [x] `internal/store/redis/client.go` - NewClient(addr) 函数
- [x] `cmd/server/main.go` - 初始化 Redis 连接，注入到 TaskManager

---

## Phase 2：重构 TaskManager

### 2.1 新增 JobRuntime 类型

新建 `internal/task/runtime.go`：

```go
type JobRuntime struct {
    TaskID         string
    WorkDir        string                    // 需要用于聚合
    OutputPath     string                    // 最终结果路径
    TotalShards    int
    SubTasks       map[string]*SubTaskMeta
    CompletedCount int
    FailedCount    int
    FailedTasks    []string

    mu            sync.Mutex
    aggregateOnce sync.Once
}

func NewJobRuntime(taskID, workDir string, totalShards int) *JobRuntime
func (jr *JobRuntime) OnSubTaskComplete(signal *CompletionSignal) error
func (jr *JobRuntime) IsAllDone() bool
func (jr *JobRuntime) Aggregate() error
```

### 2.2 修改 TaskManager 结构

修改 `internal/task/manager.go`：

```go
type TaskManager struct {
    store      store.Store              // 新增：Redis store
    activeJobs map[string]*JobRuntime   // 替换 tasks
    mu         sync.RWMutex
    pool       *worker.WorkerPool
    config     llm.Config
    stopChan   chan struct{}
}

func NewTaskManager(workerCount int, config llm.Config, store store.Store) (*TaskManager, error)
```

### 2.3 修改各方法

#### CreateTask(pdfPath) (taskID, error)

```go
// 1. 生成 taskID, workDir
// 2. 切分 PDF，创建 SubTaskMeta
// 3. 创建 TaskRecord，保存到 Redis（status=pending）
// 4. 创建 JobRuntime，加入 activeJobs
// 5. 返回 taskID
```

#### SubmitTaskToPool(taskID, timeout) error

```go
// 1. 从 activeJobs 获取 JobRuntime（不查 Redis）
// 2. 提交所有 SubTask 到 pool
// 3. 更新 Redis status → processing
```

#### handleResult(signal) error

```go
// 1. 从 activeJobs 获取 JobRuntime
// 2. 调用 jr.OnSubTaskComplete(signal)
// 3. if jr.IsAllDone():
//      go func() {
//          jr.Aggregate()
//          store.UpdateTaskStatus(id, completed)
//          delete(activeJobs, id)  // 清理内存
//      }()
```

#### GetTask(taskID) (*store.TaskRecord, error)

```go
// 直接从 Redis 读取，不查内存
return tm.store.GetTask(ctx, taskID)
```

#### WaitForTask(taskID, timeout) error

```go
// 轮询 Redis 状态（或保留轮询内存状态）
// 推荐：轮询 Redis，因为状态已经同步
```

---

## Phase 3：清理旧代码

### 3.1 删除/重构文件

- [ ] `internal/task/parent.go` - 聚合逻辑移到 JobRuntime，删除此文件
- [ ] `internal/task/types.go` - 删除 ParentTask 类型，保留 SubTaskMeta 和常量

### 3.2 更新 API handlers

- [ ] `internal/api/handlers.go` - GetTask 返回 store.TaskRecord

---

## Phase 4：启动恢复（可选，后续优化）

- [ ] `TaskManager.Start()` 时扫描 Redis 中 status=processing 的任务
- [ ] 将这些任务标记为 failed（error="服务重启中断"）
- [ ] 或：实现任务重新入队逻辑

---

## 生命周期对照表

| 操作 | Redis | activeJobs |
|------|-------|------------|
| CreateTask | SaveTask(pending) | 创建 JobRuntime |
| SubmitTaskToPool | UpdateStatus(processing) | - |
| SubTask 完成 | - | 更新计数 |
| 全部完成 | UpdateStatus(completed) | 删除 JobRuntime |
| 查询状态 | GetTask() | - |
| 删除任务 | DeleteTask() | 删除 JobRuntime（如果存在）|
| 服务重启 | 数据保留 | 丢失（processing 任务需处理）|

---

## 注意事项

1. **错误处理**：Redis 写入失败时的回滚策略
2. **并发安全**：activeJobs 的锁保护
3. **TTL**：TaskRecord 设置合理的过期时间（如 7 天）
4. **key 格式**：统一使用 `task:{id}` 前缀
