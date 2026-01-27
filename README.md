# LLM-PDF-OCR

基于 Go + 多模态LLM 的高效、经济并且准确 PDF 转 Markdown 服务，通过分片并发处理突破大文件限制。

## 💡 项目初衷

### 为什么做这个项目？

1. **Markdown 比 PDF 更适合阅读和处理**
   - 对人类：支持全文检索、版本控制、直接编辑
   - 对 LLM：纯文本格式减少 token 消耗，上下文理解更准确

2. **LLM自身的局限性**
   - Max Output Token和Max Input Token严重不对等 (以gemini-3-flash-preview为例: 最大输入Token 1m, 输出只有64k)
   - 大部分模型以阶梯式计费, 成本随着(输入/输出)Token数增长而大幅增加

3. **上下文无关的分片策略更经济**
   - OCR 任务天然适合分片：每页识别互不依赖
   - 分片后单请求 token 消耗更少，成本更低
   - 失败时只需重试单个分片，不必重跑全文

4. **充分利用并发提升效率**
   - 100 页 PDF：串行需 100 秒，5 worker 并发仅需 20 秒 (理论上)
   - Worker Pool 模式充分榨取 API 配额和并行程度 (多API支持正在路上)

## 🏗 系统架构

### 核心流程

```
用户上传 PDF
    ↓
分片处理（5页/片）
    ↓
任务队列 (buffered channel)
    ↓
Worker Pool (5 workers)
    ├─ Worker 1 → Gemini API → 分片结果1
    ├─ Worker 2 → Gemini API → 分片结果2
    └─ ...
    ↓
聚合器（按页码排序）
    ↓
输出 Markdown 文件
```

### 技术栈

-   **Go**：原生并发支持，高效的 goroutine 和 channel
-   **Gemini 3 Flash**：直接支持 PDF 上传，处理速度快、成本低
-   **pdfcpu**：纯 Go 实现的 PDF 处理库，无需 C 依赖

### 关键设计

#### 两层任务模型

```go
ParentTask (用户视角)
  ├─ ID: task_abc
  ├─ 状态: pending → processing → completed
  ├─ 进度: 15/20 (已完成/总分片数)
  └─ 结果: output/task_abc.md

SubTask (内部实现)
  ├─ 对应 PDF 的一个分片（如第1-5页）
  ├─ 完全并发执行，无顺序依赖
  └─ 聚合时按页码排序
```

#### Worker Pool 并发控制 (目前)

- **固定 worker 数量**：5 个 goroutine 并发处理
- **有界任务队列**：容量 100，防止内存爆炸
- **Rate Limiter**：精确控制 API QPM，避免触发限流
- **优雅重试**：单片失败最多重试 3 次，指数退避

#### 错误处理策略

- **部分失败容忍**：某个分片失败 3 次后跳过，用错误信息占位
- **结果完整性优先**：返回不完整但可用的 Markdown（标注缺失页码）
- **实时进度反馈**：`GET /tasks/{id}` 返回 `completed: 15, failed: 2, total: 20`

## 📂 项目结构

```bash
.
├── cmd/
│   ├── ocr-demo/           # PDF 转 Markdown 命令行工具
│   ├── gemini-demo/        # Gemini API 功能演示
│   ├── mineru-demo/        # MinerU 集成示例
│   └── server/             # HTTP 服务入口（开发中）
├── internal/
│   ├── task/               # 任务管理 (TaskManager/ParentTask/SubTask)
│   │   ├── manager.go      # 任务调度和生命周期管理
│   │   ├── parent.go       # 父任务聚合逻辑
│   │   └── types.go        # 类型定义
│   └── worker/             # Worker Pool 实现
│       ├── pool.go         # 并发任务处理
│       └── types.go        # Worker 类型定义
├── pkg/
│   ├── pdf/                # PDF 分片工具
│   ├── LLM/gemini/         # Gemini SDK 封装
│   └── result/             # 结果处理工具
└── output/                 # Markdown 输出目录
```

## 🚀 快速开始

### 环境配置

创建 `.env` 文件：
```bash
GEMINI_API_KEY=your_api_key_here
```

### 运行示例

```bash
# 安装依赖
go mod download

# 处理 PDF 文件（完整流程）
go run ./cmd/ocr-demo/main.go ./path/to/your.pdf
# 输出：./output/{task_id}/result.md

# 运行 Gemini API 演示
go run ./cmd/gemini-demo/main.go

# 运行服务（开发中）
go run ./cmd/server/main.go
```

## 🎯 开发路线图

- [x] **Phase 1**: 基础 PDF 处理和 Gemini API 集成
- [x] **Phase 2**: PDF 分片功能
- [x] **Phase 3**: Worker Pool 并发调度 + TaskManager
- [ ] **Phase 4**: HTTP API 服务
- [ ] **Phase 5**: LRU 缓存和文件管理

## 📊 性能优势 (计划中)


*测试环境：Gemini 2.5 Flash，平均单页处理 1 秒*

## 🔧 技术亮点

- **零依赖部署**：纯 Go 实现，编译为单一二进制文件
- **内存安全**：流式处理 PDF，避免大文件一次性加载
- **可观测性**：实时任务进度、详细错误日志
- **弹性设计**：部分失败不阻塞整体结果输出