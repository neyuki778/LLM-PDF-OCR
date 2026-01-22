# LLM-PDF-OCR

本项目是一个高性能 PDF OCR 服务。 **核心逻辑**：大(多) PDF → 逻辑切分 → 并发请求 Gemini API → Markdown 聚合。

## 🛠 技术选型

-   **Go 1.22+**：利用高效的协程和 `Context` 治理。
    
-   **Gemini 3 Flash**：支持直接上传 PDF，处理速度极快且成本低。
    
-   **pdfcpu / go-pdfium**：用于本地快速切分 PDF 页面。
    
-   **Redis (Task Queue)**：处理异步状态和结果缓存。
    

## 🏗 核心架构流

1.  **API 层**：接收上传，生成 `task_id`，丢入队列。
    
2.  **切分层 (Sharding)**：将大 PDF 物理/逻辑切分为单页。
    
3.  **调度层 (Worker Pool)**：控制并发数（防止 OOM 和 API 限流）。
    
4.  **API 层 (LLM)**：调用 Gemini 进行 OCR 识别。
    
5.  **存储层**：结果暂存 Redis，状态通过 Webhook 或轮询返回。
    

___

## 📅 开发计划 (Milestones)

### Phase 1: 核心链路跑通 (MVP)

-    搭建 Go 基本目录结构 (`cmd`, `internal`, `pkg`)。
    
-    实现 `pkg/pdf`：使用 `pdfcpu` 提取 PDF 的单页 `[]byte`。
    
-    实现 `pkg/llm`：调用 Gemini API (采用最新的 `application/pdf` 直接上传模式)。
    
-    **目标**：一个 CLI 命令处理一个 PDF 并输出文本。
    

### Phase 2: 并发调度 (The "Go" Power)

-    实现有界工作池 (`Worker Pool`)。
    
-    引入 `golang.org/x/sync/errgroup` 处理并发错误捕获。
    
-    引入 `golang.org/x/time/rate` 实现令牌桶限流。
    
-    **目标**：能同时调用 N 个 API 处理长文档，速度提升 5-10 倍。
    

### Phase 3: 异步化与持久化

-    集成 `Gin` 框架。
    
-    使用 Redis 存储任务状态 (`pending`, `processing`, `done`, `failed`)。
    
-    实现结果的 MD5 缓存（秒传逻辑）。
    

___

## 📂 极简目录

Plaintext

```bash
.
├── cmd/server/main.go      # 程序入口
├── internal/
│   ├── worker/             # 核心：并发调度逻辑 (Worker Pool)
│   ├── service/            # 业务：OCR 处理流水线
│   └── store/              # 存储：Redis 交互
├── pkg/
│   ├── pdf/                # 工具：PDF 拆分封装
│   └── gemini/             # 适配：Gemini API Client
└── .env                    # 配置：API_KEY, REDIS_ADDR
```

___

## 📝 开发备忘录 (Interview Points)

-   **为什么切分？** 虽然 Gemini 支持 1000 页，但并发调用 1000 个单页请求比等一个 1000 页的请求快得多，且能实现页级别的重试。
    
-   **限流策略**：Gemini 2.5 Flash 免费层级有限制，代码中必须有严格的速率控制。
    
-   **内存安全**：处理大文件时，尽量使用 `io.Reader` 流式传递，避免一次性读入整个 PDF 到内存。