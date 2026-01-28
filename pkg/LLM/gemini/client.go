package gemini

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

const (
	DefaultModel = "gemini-3-flash-preview"
	Prompt       = "Extract the PDF content and convert it into a clean Markdown format. Output only the content of the PDF without any additional commentary or preamble. Maintain the original language of the document; do not translate."
)

// Client 封装 Gemini API 客户端，实现 PDFProcessor 接口
type Client struct {
	client *genai.Client
	model  string
}

// NewClient 创建 Gemini 客户端
func NewClient(apiKey, model string) (*Client, error) {
	if model == "" {
		model = DefaultModel
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil) // API key 从环境变量 GEMINI_API_KEY 读取
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

// ProcessPDF 实现 PDFProcessor 接口，读取本地 PDF 并调用 Gemini OCR
func (c *Client) ProcessPDF(ctx context.Context, pdfPath string) (string, error) {
	// 1. 读取 PDF 文件
	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF file: %w", err)
	}

	// 2. 构建请求内容
	pdfPart := []*genai.Part{
		{
			InlineData: &genai.Blob{
				MIMEType: "application/pdf",
				Data:     pdfBytes,
			},
		},
		genai.NewPartFromText(Prompt),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(pdfPart, genai.RoleUser),
	}

	// 3. 调用 Gemini API
	result, err := c.client.Models.GenerateContent(ctx, c.model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return result.Text(), nil
}
