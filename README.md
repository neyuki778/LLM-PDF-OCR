# LLM-PDF-OCR

基于 Go 的高并发 PDF 文档处理服务。采用 **分片 + Worker Pool** 并发调度架构，将大文件拆解为独立子任务并行处理，在受上游 API 限流约束下实现接近线性的吞吐扩展。

> 线上体验地址：**https://pdf.neyuki.xyz**
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

![](./docs/架构图.svg)

### 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| HTTP 服务 | Gin | RESTful API、静态文件服务 |
| 并发调度 | goroutine + channel | Worker Pool、有界任务队列 |
| PDF 处理 | pdfcpu | 纯 Go 实现，无 CGO 依赖 |
| OCR 后端 | Gemini / MinerU | 接口抽象，支持多后端切换 |
| 持久化 | Redis + SQLite | Redis 存任务与历史索引，SQLite 存用户与 refresh token |
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

ParentTask 面向 API 层暴露整体进度；SubTask 是调度的最小单元。全部 SubTask 完成后按页码排序聚合为最终 Markdown。

### 鉴权与权限控制

- **JWT 登录体系**：邮箱注册登录，Access + Refresh 双 token。
- **Refresh Rotation**：refresh 成功后立即轮换，旧 token 作废。
- **Cookie 会话**：HttpOnly + SameSite，前端自动 refresh。
- **Owner 校验**：`GET /api/tasks/:id` 和 `GET /api/tasks/:id/result` 仅 owner 可访问。

### 分级额度（Quota）

- 按登录态区分游客/用户额度（默认 20 / 40 页）。
- `TASK_MAX_PAGES_HARD` 作为系统级兜底上限。
- access 缺失但 refresh 存在时返回 401，前端可触发 refresh 后重试。

### 持久化与历史任务

```
查询请求 → 内存 Map (活跃任务) → Redis (任务元数据) → 404
                 │
                 └── user_tasks:{user_id} (ZSET 历史索引)
```

- 创建任务时写入 `owner_user_id`。
- 登录用户可通过 `/api/tasks/history` 查看历史任务。
- 历史索引默认只保留最近 2000 条，防止无限增长。

### MinerU 图片链路增强

```
MinerU ZIP 结果 → 提取 images/* 到 output/{task_id}/images/
              → 聚合 Markdown
              → 将 images/ 相对路径重写为 /output/{task_id}/images/ 绝对可访问地址
```

- 在当前实现中，Gemini 路径以文本提取为主；MinerU 路径可返回独立图片资源。
- 任务聚合后会自动执行图片链接重写，用户下载的 Markdown 可直接渲染图片。

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
├── auth/         # 认证层：用户、JWT、refresh token、SQLite store
├── task/         # 调度层：TaskManager、ParentTask、SubTask、状态机
├── worker/       # 并发层：Worker Pool、有界队列、重试策略
└── store/redis/  # Redis 任务存储与历史索引

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

### 方式一：Docker（推荐）

无需本地安装 Go / Redis，一条命令启动全部服务。

```bash
# 1) 复制环境变量模板并填写 API key 等配置
cp .env.example .env

# 2) 构建镜像并启动（首次会自动 build）
docker compose up -d

# 3) 查看日志
docker compose logs -f app

# 4) 访问
# Web: http://localhost:8080
# API: http://localhost:8080/api/tasks
```

**重新部署 / 代码更新后：**

```bash
docker compose up -d --build
```

---

### 方式二：本地开发（需要 Go 1.22+）

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

### `.env.example` 关键配置（精简）

```bash
# OCR
LLM_PROVIDER=mineru                  # gemini | mineru
GEMINI_API_KEY=...
MINERU_TOKEN=...
PUBLIC_URL=https://your-domain

# Redis
REDIS_ADDRESS=localhost:6677

# Auth（JWT_SECRET 非空时启用）
JWT_SECRET=...
SQLITE_PATH=./data/app.db
AUTH_COOKIE_SECURE=false

# Quota
TASK_MAX_PAGES_GUEST=20
TASK_MAX_PAGES_USER=40
TASK_MAX_PAGES_HARD=100
```

## 🌐 HTTP API

### Task

| 方法 | 端点 | 说明 |
|------|------|------|
| `POST` | `/api/tasks` | 上传 PDF，创建任务，返回 `task_id` |
| `GET` | `/api/tasks/history` | 当前登录用户历史任务 |
| `GET` | `/api/tasks/:id` | 查询任务状态与进度（owner 校验） |
| `GET` | `/api/tasks/:id/result` | 获取 Markdown 文件（owner 校验） |

### Auth

| 方法 | 端点 | 说明 |
|------|------|------|
| `POST` | `/api/auth/register` | 邮箱注册 |
| `POST` | `/api/auth/login` | 登录并下发 cookie |
| `POST` | `/api/auth/refresh` | 刷新 access/refresh |
| `POST` | `/api/auth/logout` | 登出并清理 cookie |
| `GET` | `/api/auth/me` | 获取当前登录用户 |

```bash
# 上传
curl -X POST -F "file=@document.pdf" http://localhost:8080/api/tasks

# 查询进度
curl http://localhost:8080/api/tasks/{task_id}

# 获取结果
curl http://localhost:8080/api/tasks/{task_id}/result
```
