package gemini

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

const (
	ModelName = "gemini-3-flash-preview"
	// Prompt    = "识别pdf并转换成合适的markdown格式, 除了PDF内容外不要有任何额外的语句, 保持PDF原本的语言, 不要擅自翻译"
	Prompt = "Extract the PDF content and convert it into a clean Markdown format. Output only the content of the PDF without any additional commentary or preamble. Maintain the original language of the document; do not translate."
)

// ProcessPDF 读取本地 PDF 并调用 Gemini OCR，返回 Markdown 内容
func ProcessPDF(ctx context.Context, client *genai.Client, pdfPath string) (string, error) {
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
	result, err := client.Models.GenerateContent(ctx, ModelName, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return result.Text(), nil
}
