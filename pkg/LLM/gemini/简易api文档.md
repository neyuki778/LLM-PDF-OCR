# Gemini API 客户端文档

## 概述

本包提供了 Google Gemini API 的 Go 语言客户端封装，支持文本生成、流式响应和多模态（图片+文本）处理。适用于 API 网关、聊天应用和 OCR 服务等场景。

### 主要特性

- ✅ 代理支持（适配中国大陆网络环境）
- ✅ 流式生成（降低首字延迟 TTFB）
- ✅ 多模态处理（图片+文本）
- ✅ Context 取消传播
- ✅ 可配置的模型参数

---

## 快速开始

### 依赖安装

```bash
go get github.com/google/generative-ai-go/genai
go get google.golang.org/api/option
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "yourproject/pkg/LLM/gemini"
)

func main() {
    ctx := context.Background()
    
    // 创建客户端（使用代理）
    client, err := gemini.NewGeminiClient(
        ctx,
        "YOUR_API_KEY",
        "http://127.0.0.1:7890", // 代理地址，不需要代理传空字符串
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 使用客户端进行流式生成
    err = client.StreamGenerate(ctx, "介绍一下 Go 语言")
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## API 参考

### GeminiClient

客户端结构体，封装了 Gemini API 的核心功能。

```go
type GeminiClient struct {
    client *genai.Client
    model  *genai.GenerativeModel
}
```

#### NewGeminiClient

创建并初始化 Gemini 客户端。

**函数签名**

```go
func NewGeminiClient(ctx context.Context, apiKey string, proxyAddr string) (*GeminiClient, error)
```

**参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ctx` | `context.Context` | 上下文对象 |
| `apiKey` | `string` | Google AI Studio API 密钥 |
| `proxyAddr` | `string` | 代理地址（格式：`http://host:port`），不需要代理传空字符串 |

**返回值**

- `*GeminiClient`: 客户端实例
- `error`: 初始化错误

**示例**

```go
client, err := NewGeminiClient(ctx, "YOUR_API_KEY", "http://127.0.0.1:7890")
if err != nil {
    return err
}
defer client.Close()
```

**完整实现**

```go
func NewGeminiClient(ctx context.Context, apiKey string, proxyAddr string) (*GeminiClient, error) {
    opts := []option.ClientOption{
        option.WithAPIKey(apiKey),
    }

    // 自定义 HTTP Client 以处理代理
    if proxyAddr != "" {
        proxyURL, err := url.Parse(proxyAddr)
        if err != nil {
            return nil, fmt.Errorf("invalid proxy url: %w", err)
        }
        
        httpClient := &http.Client{
            Transport: &http.Transport{
                Proxy: http.ProxyURL(proxyURL),
            },
        }
        opts = append(opts, option.WithHTTPClient(httpClient))
    }

    client, err := genai.NewClient(ctx, opts...)
    if err != nil {
        return nil, err
    }

    // 默认使用 gemini-1.5-flash
    model := client.GenerativeModel("gemini-1.5-flash")
    model.SetTemperature(0.7)

    return &GeminiClient{
        client: client,
        model:  model,
    }, nil
}
```

---

#### Close

关闭客户端连接，释放资源。

**函数签名**

```go
func (g *GeminiClient) Close()
```

**示例**

```go
defer client.Close()
```

---

#### StreamGenerate

流式生成文本内容，适用于聊天应用和 API 网关场景。

**函数签名**

```go
func (g *GeminiClient) StreamGenerate(ctx context.Context, prompt string) error
```

**参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ctx` | `context.Context` | 上下文对象，用于取消请求 |
| `prompt` | `string` | 用户输入的提示词 |

**返回值**

- `error`: 流式处理错误

**使用场景**

- API 网关：降低首字延迟（TTFB）
- 聊天应用：实时响应用户输入
- 长文本生成：避免请求超时

**示例**

```go
err := client.StreamGenerate(ctx, "写一首关于春天的诗")
if err != nil {
    log.Printf("stream error: %v", err)
}
```

**完整实现**

```go
func (g *GeminiClient) StreamGenerate(ctx context.Context, prompt string) error {
    iter := g.model.GenerateContentStream(ctx, genai.Text(prompt))
    
    for {
        resp, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return fmt.Errorf("stream error: %w", err)
        }

        // 处理分块响应
        for _, cand := range resp.Candidates {
            if cand.Content != nil {
                for _, part := range cand.Content.Parts {
                    if txt, ok := part.(genai.Text); ok {
                        // 推送到 WebSocket 或 HTTP Response Writer
                        fmt.Print(string(txt))
                    }
                }
            }
        }
    }
    return nil
}
```

---

#### AnalyzeImage

分析图片内容，支持多模态处理。

**函数签名**

```go
func (g *GeminiClient) AnalyzeImage(ctx context.Context, imgData []byte, prompt string) (string, error)
```

**参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ctx` | `context.Context` | 上下文对象 |
| `imgData` | `[]byte` | 图片二进制数据 |
| `prompt` | `string` | 对图片的提问或指令 |

**返回值**

- `string`: 分析结果文本
- `error`: 处理错误

**支持格式**

- PNG
- JPEG
- WEBP
- HEIC

**使用场景**

- OCR 服务：提取图片中的文字
- 图片描述：生成图片的文字描述
- 视觉问答：基于图片内容回答问题

**示例**

```go
imgData, err := os.ReadFile("document.jpg")
if err != nil {
    return err
}

result, err := client.AnalyzeImage(ctx, imgData, "提取图片中的所有文字")
if err != nil {
    return err
}
fmt.Println(result)
```

**完整实现**

```go
func (g *GeminiClient) AnalyzeImage(ctx context.Context, imgData []byte, prompt string) (string, error) {
    resp, err := g.model.GenerateContent(ctx, 
        genai.Text(prompt),
        genai.ImageData("image/jpeg", imgData)) // 根据实际类型修改
    
    if err != nil {
        return "", err
    }

    if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
        return "", fmt.Errorf("no content generated")
    }

    // 提取文本结果
    var result string
    for _, part := range resp.Candidates[0].Content.Parts {
        if txt, ok := part.(genai.Text); ok {
            result += string(txt)
        }
    }
    return result, nil
}
```

---

## 模型选择

### gemini-1.5-flash（推荐）

**特点**
- ⚡️ 极快的响应速度
- 💰 成本低廉
- ✅ 适合高频 API 调用
- ✅ 适合简单的逻辑任务

**适用场景**
- API 网关默认模型
- 聊天机器人
- 文本摘要
- 简单的内容生成

### gemini-1.5-pro

**特点**
- 🧠 强大的推理能力
- 📊 适合复杂逻辑分析
- ⏱ 响应速度稍慢

**适用场景**
- 代码分析
- 深度问答
- 复杂的内容理解
- 多步骤推理任务

---

## 最佳实践

### 1. Context 管理

所有 API 调用都接受 `context.Context` 参数。在 Gateway 或 Web 应用中，务必传入 HTTP Request 的 context，以实现：

- ✅ 客户端断开时自动取消 Gemini 请求
- ✅ 节省 Token 用量
- ✅ 避免资源浪费

**示例**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // 使用请求的 context
    err := client.StreamGenerate(r.Context(), prompt)
    if err != nil {
        // 处理错误
    }
}
```

### 2. 错误处理

注意捕获 `googleapi.Error`，其中包含重要的 HTTP 状态信息：

| 状态码 | 说明 | 处理建议 |
|--------|------|----------|
| `429` | Quota Exceeded | 实施限流或指数退避重试 |
| `400` | Bad Request | 检查请求参数 |
| `401` | Unauthorized | 验证 API Key |
| `500` | Internal Server Error | 重试请求 |

**示例**

```go
import "google.golang.org/api/googleapi"

if err != nil {
    if apiErr, ok := err.(*googleapi.Error); ok {
        switch apiErr.Code {
        case 429:
            // 实施退避策略
            time.Sleep(time.Second * 5)
            // 重试请求
        case 401:
            // API Key 无效
            return fmt.Errorf("invalid API key")
        }
    }
    return err
}
```

### 3. 代理配置

在中国大陆访问 Google API 时，通常需要配置代理：

```go
// 开发环境
client, _ := NewGeminiClient(ctx, apiKey, "http://127.0.0.1:7890")

// 生产环境（从环境变量读取）
proxyAddr := os.Getenv("HTTP_PROXY")
client, _ := NewGeminiClient(ctx, apiKey, proxyAddr)
```

### 4. 资源释放

始终在使用完客户端后调用 `Close()` 方法：

```go
client, err := NewGeminiClient(ctx, apiKey, proxyAddr)
if err != nil {
    return err
}
defer client.Close() // 确保资源被释放
```

---

## 完整示例

### 示例 1: 简单文本生成

```go
package main

import (
    "context"
    "log"
    "yourproject/pkg/LLM/gemini"
)

func main() {
    ctx := context.Background()
    client, err := gemini.NewGeminiClient(ctx, "YOUR_API_KEY", "")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    err = client.StreamGenerate(ctx, "解释一下什么是 RESTful API")
    if err != nil {
        log.Fatal(err)
    }
}
```

### 示例 2: OCR 图片识别

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "yourproject/pkg/LLM/gemini"
)

func main() {
    ctx := context.Background()
    client, err := gemini.NewGeminiClient(
        ctx,
        os.Getenv("GEMINI_API_KEY"),
        os.Getenv("HTTP_PROXY"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 读取图片
    imgData, err := os.ReadFile("invoice.png")
    if err != nil {
        log.Fatal(err)
    }

    // 提取发票信息
    result, err := client.AnalyzeImage(
        ctx,
        imgData,
        "提取这张发票的所有字段信息，包括日期、金额、购买方、销售方",
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result)
}
```

### 示例 3: HTTP API 集成

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "yourproject/pkg/LLM/gemini"
)

var client *gemini.GeminiClient

func handler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Prompt string `json:"prompt"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 使用请求的 context，客户端断开时自动取消
    err := client.StreamGenerate(r.Context(), req.Prompt)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func main() {
    var err error
    client, err = gemini.NewGeminiClient(
        context.Background(),
        "YOUR_API_KEY",
        "http://127.0.0.1:7890",
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()

    http.HandleFunc("/generate", handler)
    http.ListenAndServe(":8080", nil)
}
```

---

## 故障排查

### 问题 1: 请求超时

**原因**: 未配置代理或代理地址错误

**解决方案**:
```go
// 检查代理是否可用
client, err := NewGeminiClient(ctx, apiKey, "http://127.0.0.1:7890")
```

### 问题 2: API Key 无效

**原因**: API Key 错误或未授权

**解决方案**:
1. 访问 [Google AI Studio](https://makersuite.google.com/app/apikey) 获取 API Key
2. 确保 API Key 已启用 Gemini API 访问权限

### 问题 3: 响应为空

**原因**: 提示词触发内容安全过滤

**解决方案**:
- 调整提示词内容
- 检查响应中的 `SafetyRatings` 字段

---

## 相关资源

- [Gemini API 官方文档](https://ai.google.dev/docs)
- [Go SDK GitHub 仓库](https://github.com/google/generative-ai-go)
- [Google AI Studio](https://makersuite.google.com/)