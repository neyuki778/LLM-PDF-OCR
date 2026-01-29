package mineru

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neyuki778/LLM-PDF-OCR/pkg/result"
)

type Client struct {
	BaseURL   string // MinerU API 地址，如 https://mineru.net
	Token     string // MinerU API Token
	PublicURL string // 本服务的公开地址，如 https://yourdomain.com
	HTTP      *http.Client
}

func NewClient(baseURL, token, publicURL string) *Client {
	if baseURL == "" {
		baseURL = "https://mineru.net"
	}
	return &Client{
		BaseURL:   baseURL,
		Token:     token,
		PublicURL: publicURL,
		HTTP:      &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*CreateTaskResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/api/v4/extract/task"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != 0 {
		return &out, fmt.Errorf("mineru create failed: code=%d msg=%s trace_id=%s", out.Code, out.Msg, out.TraceID)
	}
	return &out, nil
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*GetTaskResponse, error) {
	url := c.BaseURL + "/api/v4/extract/task/" + taskID
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Accept", "*/*")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out GetTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != 0 {
		return &out, fmt.Errorf("mineru get failed: code=%d msg=%s trace_id=%s", out.Code, out.Msg, out.TraceID)
	}
	return &out, nil
}

func (c *Client) WaitForCompletion(ctx context.Context, taskID string) (*GetTaskResponse, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task: %w", ctx.Err())
		case <-ticker.C:
			task, err := c.GetTask(ctx, taskID)
			if err != nil {
				return nil, err
			}

			fmt.Printf("Progress: %d/%d pages (state: %s)\n",
				task.Data.ProgressInfo.ExtractedPages,
				task.Data.ProgressInfo.TotalPages,
				task.Data.State)

			switch task.Data.State {
			case "done":
				return task, nil
			case "failed":
				return nil, fmt.Errorf("task failed: %s", task.Data.ErrMsg)
			}
		}
	}
}

// ProcessPDF 实现 PDFProcessor 接口
// pdfPath 是本地文件路径，如 uploads/xxx.pdf
func (c *Client) ProcessPDF(ctx context.Context, pdfPath string) (string, error) {
	// 1. 将本地路径转换为公开 URL
	// pdfPath 格式: output/<task_id>/xxx.pdf -> https://yourdomain.com/output/<task_id>/xxx.pdf
	cleanPath := filepath.Clean(pdfPath)
	outputPrefix := filepath.Clean("output") + string(os.PathSeparator)
	if !strings.HasPrefix(cleanPath, outputPrefix) {
		return "", fmt.Errorf("pdf path must be under output/: %s", pdfPath)
	}
	relative := strings.TrimPrefix(cleanPath, outputPrefix)
	publicBase := strings.TrimRight(c.PublicURL, "/")
	pdfURL := publicBase + "/output/" + filepath.ToSlash(relative)

	// 2. 创建 MinerU 任务
	createResp, err := c.CreateTask(ctx, CreateTaskRequest{
		URL:          pdfURL,
		ModelVersion: "vlm",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create mineru task: %w", err)
	}

	// 3. 等待任务完成
	taskResp, err := c.WaitForCompletion(ctx, createResp.Data.TaskID)
	if err != nil {
		return "", fmt.Errorf("failed to wait for mineru task: %w", err)
	}

	// 4. 下载结果 ZIP
	tempDir := os.TempDir()
	zipPath := filepath.Join(tempDir, createResp.Data.TaskID+".zip")
	if err := result.DownloadZip(ctx, taskResp.Data.FullZipURL, zipPath); err != nil {
		return "", fmt.Errorf("failed to download result: %w", err)
	}
	defer os.Remove(zipPath)

	// 5. 提取 Markdown 内容
	markdown, err := result.ExtractMarkdown(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract markdown: %w", err)
	}

	return markdown, nil
}
