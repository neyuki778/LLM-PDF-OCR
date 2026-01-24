# 文档理解

Gemini 模型可以处理 PDF 格式的文档，并使用原生视觉功能来理解整个文档的上下文。这不仅仅是提取文本，还让 Gemini 能够：

-   分析和解读内容，包括文本、图片、图表、图表和表格，即使是长达 1,000 页的文档也能轻松应对。
-   以[结构化输出](https://ai.google.dev/gemini-api/docs/structured-output?hl=zh-cn)格式提取信息。
-   根据文档中的视觉和文本元素总结内容并回答问题。
-   转写文档内容（例如转写为 HTML），同时保留布局和格式，以便在下游应用中使用。

您也可以通过相同的方式传递非 PDF 文档，但 Gemini 会将这些文档视为普通文本，从而消除图表或格式等上下文。

## 以内嵌方式传递 PDF 数据

您可以在向 `generateContent` 发出的请求中内嵌传递 PDF 数据。此方法最适合处理较小的文档或临时处理，因为您无需在后续请求中引用该文件。对于需要在多轮对话中参考的较大文档，我们建议使用 [Files API](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn#large-pdfs)，以缩短请求延迟时间并减少带宽使用量。

以下示例展示了如何从网址提取 PDF 并将其转换为字节以进行处理：

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[JavaScript](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#javascript)[Go](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#go)[REST](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#rest)

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "google.golang.org/genai"
)

func main() {

    ctx := context.Background()
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  os.Getenv("GEMINI_API_KEY"),
        Backend: genai.BackendGeminiAPI,
    })

    pdfResp, _ := http.Get("https://discovery.ucl.ac.uk/id/eprint/10089234/1/343019_3_art_0_py4t4l_convrt.pdf")
    var pdfBytes []byte
    if pdfResp != nil && pdfResp.Body != nil {
        pdfBytes, _ = io.ReadAll(pdfResp.Body)
        pdfResp.Body.Close()
    }

    parts := []*genai.Part{
        &genai.Part{
            InlineData: &genai.Blob{
                MIMEType: "application/pdf",
                Data:     pdfBytes,
            },
        },
        genai.NewPartFromText("Summarize this document"),
    }

    contents := []*genai.Content{
        genai.NewContentFromParts(parts, genai.RoleUser),
    }

    result, _ := client.Models.GenerateContent(
        ctx,
        "gemini-3-flash-preview",
        contents,
        nil,
    )

    fmt.Println(result.Text())
}
```

您还可以从本地文件读取 PDF 以进行处理：

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[JavaScript](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#javascript)[Go](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#go)

```css
package main

import (
    "context"
    "fmt"
    "os"
    "google.golang.org/genai"
)

func main() {

    ctx := context.Background()
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  os.Getenv("GEMINI_API_KEY"),
        Backend: genai.BackendGeminiAPI,
    })

    pdfBytes, _ := os.ReadFile("path/to/your/file.pdf")

    parts := []*genai.Part{
        &genai.Part{
            InlineData: &genai.Blob{
                MIMEType: "application/pdf",
                Data:     pdfBytes,
            },
        },
        genai.NewPartFromText("Summarize this document"),
    }
    contents := []*genai.Content{
        genai.NewContentFromParts(parts, genai.RoleUser),
    }

    result, _ := client.Models.GenerateContent(
        ctx,
        "gemini-3-flash-preview",
        contents,
        nil,
    )

    fmt.Println(result.Text())
}
```

## 使用 Files API 上传 PDF

对于较大的文件，或者当您打算在多个请求中重复使用文档时，我们建议您使用 Files API。这样可以将文件上传与模型请求分离，从而缩短请求延迟时间并减少带宽用量。

**注意**： 在已推出 Gemini API 的所有地区，Files API 均可免费使用。上传的文件会存储 48 小时。

### 来自网址的大型 PDF 文件

使用 File API 可简化通过网址上传和处理大型 PDF 文件的流程：

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[JavaScript](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#javascript)[Go](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#go)[REST](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#rest)

```go
package main

import (
  "context"
  "fmt"
  "io"
  "net/http"
  "os"
  "google.golang.org/genai"
)

func main() {

  ctx := context.Background()
  client, _ := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
  })

  pdfURL := "https://www.nasa.gov/wp-content/uploads/static/history/alsj/a17/A17_FlightPlan.pdf"
  localPdfPath := "A17_FlightPlan_downloaded.pdf"

  respHttp, _ := http.Get(pdfURL)
  defer respHttp.Body.Close()

  outFile, _ := os.Create(localPdfPath)
  defer outFile.Close()

  _, _ = io.Copy(outFile, respHttp.Body)

  uploadConfig := &genai.UploadFileConfig{MIMEType: "application/pdf"}
  uploadedFile, _ := client.Files.UploadFromPath(ctx, localPdfPath, uploadConfig)

  promptParts := []*genai.Part{
    genai.NewPartFromURI(uploadedFile.URI, uploadedFile.MIMEType),
    genai.NewPartFromText("Summarize this document"),
  }
  contents := []*genai.Content{
    genai.NewContentFromParts(promptParts, genai.RoleUser), // Specify role
  }

    result, _ := client.Models.GenerateContent(
        ctx,
        "gemini-3-flash-preview",
        contents,
        nil,
    )

  fmt.Println(result.Text())
}
```

### 本地存储的大型 PDF

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[JavaScript](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#javascript)[Go](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#go)[REST](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#rest)

```css
package main

import (
    "context"
    "fmt"
    "os"
    "google.golang.org/genai"
)

func main() {

    ctx := context.Background()
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  os.Getenv("GEMINI_API_KEY"),
        Backend: genai.BackendGeminiAPI,
    })
    localPdfPath := "/path/to/file.pdf"

    uploadConfig := &genai.UploadFileConfig{MIMEType: "application/pdf"}
    uploadedFile, _ := client.Files.UploadFromPath(ctx, localPdfPath, uploadConfig)

    promptParts := []*genai.Part{
        genai.NewPartFromURI(uploadedFile.URI, uploadedFile.MIMEType),
        genai.NewPartFromText("Give me a summary of this pdf file."),
    }
    contents := []*genai.Content{
        genai.NewContentFromParts(promptParts, genai.RoleUser),
    }

    result, _ := client.Models.GenerateContent(
        ctx,
        "gemini-3-flash-preview",
        contents,
        nil,
    )

    fmt.Println(result.Text())
}
```

您可以调用 [`files.get`](https://ai.google.dev/api/rest/v1beta/files/get?hl=zh-cn) 来验证 API 是否已成功存储上传的文件并获取其元数据。只有 `name`（以及扩展的 `uri`）是唯一的。

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[REST](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#rest)

```csharp
from google import genai
import pathlib

client = genai.Client()

fpath = pathlib.Path('example.txt')
fpath.write_text('hello')

file = client.files.upload(file='example.txt')

file_info = client.files.get(name=file.name)
print(file_info.model_dump_json(indent=4))
```

## 传递多个 PDF

Gemini API 能够在单个请求中处理多个 PDF 文档（最多 1, 000 页），前提是文档和文本提示的总大小不超过模型的上下文窗口大小。

[Python](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#python)[JavaScript](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#javascript)[Go](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#go)[REST](https://ai.google.dev/gemini-api/docs/document-processing?hl=zh-cn/#rest)

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "google.golang.org/genai"
)

func main() {

    ctx := context.Background()
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  os.Getenv("GEMINI_API_KEY"),
        Backend: genai.BackendGeminiAPI,
    })

    docUrl1 := "https://arxiv.org/pdf/2312.11805"
    docUrl2 := "https://arxiv.org/pdf/2403.05530"
    localPath1 := "doc1_downloaded.pdf"
    localPath2 := "doc2_downloaded.pdf"

    respHttp1, _ := http.Get(docUrl1)
    defer respHttp1.Body.Close()

    outFile1, _ := os.Create(localPath1)
    _, _ = io.Copy(outFile1, respHttp1.Body)
    outFile1.Close()

    respHttp2, _ := http.Get(docUrl2)
    defer respHttp2.Body.Close()

    outFile2, _ := os.Create(localPath2)
    _, _ = io.Copy(outFile2, respHttp2.Body)
    outFile2.Close()

    uploadConfig1 := &genai.UploadFileConfig{MIMEType: "application/pdf"}
    uploadedFile1, _ := client.Files.UploadFromPath(ctx, localPath1, uploadConfig1)

    uploadConfig2 := &genai.UploadFileConfig{MIMEType: "application/pdf"}
    uploadedFile2, _ := client.Files.UploadFromPath(ctx, localPath2, uploadConfig2)

    promptParts := []*genai.Part{
        genai.NewPartFromURI(uploadedFile1.URI, uploadedFile1.MIMEType),
        genai.NewPartFromURI(uploadedFile2.URI, uploadedFile2.MIMEType),
        genai.NewPartFromText("What is the difference between each of the " +
                              "main benchmarks between these two papers? " +
                              "Output these in a table."),
    }
    contents := []*genai.Content{
        genai.NewContentFromParts(promptParts, genai.RoleUser),
    }

    modelName := "gemini-3-flash-preview"
    result, _ := client.Models.GenerateContent(
        ctx,
        modelName,
        contents,
        nil,
    )

    fmt.Println(result.Text())
}
```

## 技术详情

Gemini 支持不超过 50MB 或 1,000 页的 PDF 文件。此限制适用于内嵌数据和 Files API 上传。每个文档页面相当于 258 个词元。

虽然除了模型的[上下文窗口](https://ai.google.dev/gemini-api/docs/long-context?hl=zh-cn)之外，文档中的像素数量没有具体限制，但较大的页面会被缩小到最大分辨率 (3072 x 3072)，同时保留其原始宽高比，而较小的页面会被放大到 768 x 768 像素。除了带宽之外，较低分辨率的网页不会降低费用，而较高分辨率的网页也不会提高性能。

### Gemini 3 模型

Gemini 3 通过 `media_resolution` 参数引入了对多模态视觉处理的精细控制。您现在可以为每个媒体部分单独设置低、中或高分辨率。添加此功能后，PDF 文档的处理方式已更新：

1.  **原生文本包含**：提取 PDF 中原生嵌入的文本并将其提供给模型。
2.  **结算和代币报告**：
    -   您**无需支付**因 PDF 中的提取**原生文本**而产生的令牌费用。
    -   在 API 响应的 `usage_metadata` 部分，通过处理 PDF 页面（作为图片）生成的令牌现在计入 `IMAGE` 模态，而不是像某些早期版本那样计入单独的 `DOCUMENT` 模态。

如需详细了解媒体分辨率参数，请参阅[媒体分辨率](https://ai.google.dev/gemini-api/docs/media-resolution?hl=zh-cn)指南。

### 文档类型

从技术上讲，您可以传递其他 MIME 类型以进行文档理解，例如 TXT、Markdown、HTML、XML 等。不过，文档视觉 **_仅能有意义地理解 PDF_**。其他类型的文件将被提取为纯文本，模型将无法解读我们在这些文件的呈现中看到的内容。所有特定于文件类型的信息（例如图表、示意图、HTML 标记、Markdown 格式等）都将丢失。

如需了解其他文件输入方法，请参阅[文件输入方法](https://ai.google.dev/gemini-api/docs/file-input-methods?hl=zh-cn)指南。

### 最佳做法

为了达到最佳效果，请注意以下事项：

-   请先将页面旋转到正确方向，然后再上传。
-   避免网页模糊不清。
-   如果使用单个页面，请将文本提示放在该页面之后。

## 后续步骤

如需了解详情，请参阅以下资源：

-   [文件提示策略](https://ai.google.dev/gemini-api/docs/files?hl=zh-cn#prompt-guide)：Gemini API 支持使用文本、图片、音频和视频数据进行提示，也称为多模态提示。
-   [系统指令](https://ai.google.dev/gemini-api/docs/text-generation?hl=zh-cn#system-instructions)：系统指令可让您根据自己的特定需求和使用情形来控制模型的行为。