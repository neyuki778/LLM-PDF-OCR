# Task 模块实现待办清单

## 实现顺序说明

按照依赖关系从底向上，分6个阶段实现。每个阶段完成后**先测试再进入下一阶段**。

---

## 阶段1：ParentTask基础方法 ✅

**文件**：`internal/task/parent.go`

- [ ] `NewParentTask(id, pdfPath, workDir string) *ParentTask`
  - 初始化ParentTask结构
  - 创建SubTasks map
  - 设置状态为pending

- [ ] `OnSubTaskComplete(signal *CompletionSignal)`
  - 加锁保护
  - 更新SubTaskMeta状态（success/failed）
  - 更新CompletedCount计数器
  - 记录失败任务到FailedTasks列表
  - 检测完成后调用Aggregate（用aggregateOnce保证只执行一次）

- [ ] `IsAllDone() bool`
  - 判断 `CompletedCount == TotalShards`


---

## 阶段2：TaskManager初始化和监听器

**文件**：`internal/task/manager.go`

- [ ] `NewTaskManager(workerCount int) *TaskManager`
  - 初始化tasks map
  - 创建WorkerPool
  - 创建stopChan

- [ ] `Start()`
  - 启动WorkerPool（调用pool.Start()）
  - 启动监听器goroutine（调用listenResults()）

- [ ] `listenResults()`（私有方法）
  - 循环监听pool.resultChan
  - 根据signal.ParentID查找ParentTask（加读锁）
  - 调用parentTask.OnSubTaskComplete(signal)
  - 处理stopChan退出信号

- [ ] `Shutdown()`
  - 发送stopChan信号
  - 关闭WorkerPool（调用pool.Shutdown()）

---

## 阶段3：任务创建和PDF分片

**文件**：`internal/task/manager.go`

- [ ] `CreateTask(pdfPath string) (taskID string, error)`
  1. 生成UUID作为taskID（使用github.com/google/uuid）
  2. 创建工作目录`./output/{taskID}/`
  3. 调用`pkg/pdf.SplitPDF()`切分PDF
  4. 创建ParentTask
  5. 遍历分片文件，为每个分片创建SubTaskMeta
  6. 构造worker.SubTask并提交到WorkerPool
  7. 更新ParentTask状态为processing
  8. 将ParentTask存入tasks map（加写锁）

**关键细节**：
- SubTask.PDFPath = `./output/{taskID}/split_{i}.pdf`
- SubTask.TempFilePath = `./output/{taskID}/page_{PageStart}.md`
- SubTask.MaxRetries = 3
- 提交超时时间建议5秒

---

## 阶段4：Worker实际处理逻辑

**文件**：`internal/worker/pool.go`

- [ ] 更新`worker()`方法，调用`processTask()`
- [ ] 实现`processTask(task *SubTask) error`（私有方法）
  1. 读取分片PDF文件（task.PDFPath）
  2. 调用Gemini API（参考cmd/gemini-demo/main.go的处理逻辑）
  3. 将返回的Markdown写入临时文件（task.TempFilePath）
  4. 返回错误（如果失败）

**关键细节**：
- 使用wp.geminiClient调用API
- 指数退避重试逻辑：2s, 4s, 8s
- 只在最终成功/失败时发送CompletionSignal

**需要更新的类型**：
- [ ] 更新`internal/worker/types.go`的SubTask，添加`TempFilePath string`字段

---

## 阶段5：结果聚合

**文件**：`internal/task/parent.go`

- [ ] `Aggregate() error`
  1. 创建result.md文件
  2. 按PageStart顺序遍历SubTasks
  3. 对于成功的SubTask：读取临时MD文件，追加到result.md
  4. 对于失败的SubTask：写入错误占位符
     ```markdown
     # Page {PageStart}-{PageEnd}
     [OCR失败: {Error}，已重试3次]
     ```
  5. 删除所有split_*.pdf
  6. 删除所有page_*.md
  7. 更新ParentTask状态为completed

**关键细节**：
- 聚合前加锁
- 使用sort对SubTasks按PageStart排序
- 删除文件时忽略错误（文件可能不存在）


---

## 阶段6：辅助功能（可选）

**文件**：`internal/task/manager.go`

- [ ] `GetTask(taskID string) (*ParentTask, error)`
  - 从tasks map查询（加读锁）
  - 返回ParentTask副本或错误

- [ ] `DeleteTask(taskID string) error`
  - 删除整个任务目录`./output/{taskID}/`
  - 从tasks map移除（加写锁）

---

## 依赖和工具

### 需要安装的包
```bash
go get github.com/google/uuid
```

### 需要更新的现有文件
- [ ] `internal/worker/types.go` - SubTask添加TempFilePath字段
- [ ] `internal/worker/pool.go` - 实现processTask逻辑

### 需要创建的新文件
- [ ] `internal/task/parent.go` - ParentTask方法实现
- [ ] `cmd/task-demo/main.go` - 完整流程测试程序

---

## 测试策略

每个阶段的测试用例：

```
internal/task/
├── parent_test.go       # 测试ParentTask方法
├── manager_test.go      # 测试TaskManager
└── integration_test.go  # 端到端测试
```

最小可测试demo：
```go
// cmd/task-demo/main.go
func main() {
    tm := task.NewTaskManager(5)
    tm.Start()

    taskID, _ := tm.CreateTask("test.pdf")
    fmt.Printf("任务创建: %s\n", taskID)

    // 轮询任务状态
    for {
        task, _ := tm.GetTask(taskID)
        fmt.Printf("进度: %d/%d\n", task.CompletedCount, task.TotalShards)
        if task.Status == "completed" {
            break
        }
        time.Sleep(2 * time.Second)
    }

    tm.Shutdown()
}
```

---

## 注意事项

1. **并发安全**：所有访问ParentTask和tasks map的地方都要加锁
2. **错误处理**：文件I/O和API调用都要检查错误
3. **资源清理**：确保中间文件在聚合后删除
4. **路径处理**：使用filepath.Join而不是字符串拼接
5. **UUID导入**：使用`github.com/google/uuid`而不是标准库
