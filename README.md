# LLM-PDF-OCR

基于 Go 的高并发 PDF 文档处理服务。采用 **分片 + Worker Pool** 并发调度架构，将大文件拆解为独立子任务并行处理，在受上游 API 限流约束下实现接近线性的吞吐扩展。

> 线上体验地址：**https://pdf.kana.engineer**
> 
> MinerU 路径支持图片资产提取与 Markdown 图片链接重写，下载后的 `.md` 可直接显示图片。

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
   - Worker Pool 模式充分榨取 API 配额和并行能力

## 🏗 系统架构

![](./docs/系统架构.svg)

### 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| HTTP 服务 | Gin | RESTful API、静态文件服务 |
| 并发调度 | goroutine + channel | Worker Pool、有界任务队列 |
| PDF 处理 | pdfcpu | 纯 Go 实现，无 CGO 依赖 |
| OCR 后端 | Gemini / MinerU | 接口抽象，支持多后端切换 |
| 持久化 | Redis | 任务记录缓存，TTL 自动过期 |
| 语言 | Go 1.25.4 | 编译为单一二进制，便于部署与运维 |

## 🔧 关键设计

### Producer-Consumer 并发模型

```
TaskManager (Producer)              Worker Pool (Consumer)
     │                                    │
     │  ── SubTask ──▶  [Bounded Queue]  ──▶  Worker 1 ──▶ LLM API
     │  ── SubTask ──▶  [  cap: 100   ]  ──▶  Worker 2 ──▶ LLM API
     │  ── SubTask ──▶  [             ]  ──▶  Worker 3 ──▶ LLM API
     │                                    │
     │  ◀── CompletionSignal ─────────────┘
```

- **有界队列**：容量 100，队列满时生产者阻塞，天然实现背压控制
- **固定 Worker 数**：goroutine 池化复用，避免无限制创建协程导致资源耗尽
- **CompletionSignal**：Worker 完成后通过 channel 回传信号，驱动 ParentTask 聚合

### 两层任务模型

```go
ParentTask (用户视角)                SubTask (内部调度)
├─ ID: "task_abc"                   ├─ 对应 PDF 的一个分片
├─ Status: pending → processing     ├─ 独立处理，无上下文依赖
│          → completed/failed       ├─ MaxRetries: 3
├─ Progress: 15/20                  └─ 完成后触发聚合检查
└─ Result: output/task_abc/result.md
```

ParentTask 面向 API 层暴露整体进度；SubTask 是调度的最小单元。全部 SubTask 完成后按页码排序聚合为最终 Markdown。

### 容错与降级

- **指数退避重试**：失败后等待 2^n 秒，最多 3 次，避免 API 限流雪崩
- **部分失败容忍**：单个分片重试耗尽后跳过，错误占位符标记缺失页码，其余内容正常输出
- **降级优先于失败**：返回不完整但可用的结果，而非整体报错

### 持久化策略

```
查询请求 → 内存 Map (活跃任务) → Redis (已完成任务, TTL 5h) → 404
```

- 处理中的任务存活于内存，零延迟访问
- 完成后写入 Redis 并释放内存，避免内存无限增长
- Redis 设置 TTL，自动清理过期数据

### MinerU 图片链路增强

```
MinerU ZIP 结果 → 提取 images/* 到 output/{task_id}/images/
              → 聚合 Markdown
              → 将 images/ 相对路径重写为 /output/{task_id}/images/ 绝对可访问地址
```

- 在当前实现中，Gemini 路径以文本提取为主；MinerU 路径可返回独立图片资源。
- 任务聚合后会自动执行图片链接重写，用户下载的 Markdown 可直接渲染图片。
- 图片由服务端统一托管在任务输出目录下（可通过 `PUBLIC_URL` 对外访问）。

### 接口抽象

```go
type PDFProcessor interface {
    ProcessPDF(ctx context.Context, pdfPath string) (string, error)
}
```

通过接口抽象 OCR 后端，运行时根据配置注入 Gemini 或 MinerU 实现。新增后端只需实现该接口，无需修改调度逻辑。

## 📂 项目结构

```
internal/
├── api/          # HTTP 层：路由注册、请求处理、响应序列化
├── task/         # 调度层：TaskManager、ParentTask、SubTask、状态机
└── worker/       # 并发层：Worker Pool、有界队列、重试策略

pkg/
├── LLM/          # LLM 后端抽象层
│   ├── gemini/   #   Gemini SDK 封装
│   └── MinerU/   #   MinerU REST 客户端
├── pdf/          # PDF 分片 (pdfcpu)
└── result/       # 结果处理：ZIP 下载、Markdown 提取

cmd/
├── server/       # HTTP 服务入口
└── ocr-demo/     # CLI 工具入口
```

## 🚀 快速开始

```bash
# 1) 复制环境变量模板
cp .env.example .env

# 2) 启动 Redis（必需）
docker compose up -d redis

# 3) 启动 HTTP 服务
go run ./cmd/server/main.go

# 4) 访问
# Web: http://localhost:8080
# API: http://localhost:8080/api/tasks
```

### `.env.example` 关键配置

```bash
GEMINI_API_KEY=your_gemini_api_key_here
LLM_PROVIDER=mineru                   # gemini | mineru
GEMINI_MODEL=gemini-3-flash-preview

# 仅 mineru 需要
PUBLIC_URL=https://pdf.kana.engineer  # 也可以用服务器公网 IP，例如 http://1.2.3.4:8080
MINERU_TOKEN=your_mineru_token_here
MINERU_BASE_URL=https://mineru.net
MINERU_MODEL_VERSION=vlm

# docker-compose 默认映射 6677:6379
REDIS_ADDRESS=localhost:6677
```

- `PUBLIC_URL` 仅在 `LLM_PROVIDER=mineru` 时必填，用于让 MinerU 回调/拉取可访问的 PDF 地址。
- `PUBLIC_URL` 可以是域名，也可以是服务器公网 IP + 端口（如 `http://1.2.3.4:8080`），请确保外网可访问，且不要带结尾 `/`。
- 如果你使用 `LLM_PROVIDER=gemini`，可以忽略 `PUBLIC_URL` 和 `MINERU_*` 配置。
- 当前线上地址：`https://pdf.kana.engineer`

## 🌐 HTTP API

| 方法 | 端点 | 说明 |
|------|------|------|
| `POST` | `/api/tasks` | 上传 PDF，创建任务，返回 `task_id` |
| `GET` | `/api/tasks/:id` | 查询任务状态与进度 |
| `GET` | `/api/tasks/:id/result` | 获取 Markdown 文件（任务未完成时返回状态信息） |
| `DELETE` | `/api/tasks/:id` | 删除任务（暂未实现，当前返回 501） |

```bash
# 上传
curl -X POST -F "file=@document.pdf" http://localhost:8080/api/tasks

# 查询进度
curl http://localhost:8080/api/tasks/{task_id}

# 获取结果
curl http://localhost:8080/api/tasks/{task_id}/result
```
