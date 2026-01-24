package mineru

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	result "github.com/neyuki778/LLM-PDF-OCR/pkg/result"
)

// ProcessTask 一站式处理：等待任务完成 + 下载 + 提取内容
// 这是最便利的方法，自动处理所有步骤并返回解析后的内容
func (c *Client) ProcessTask(ctx context.Context, taskID string) (*result.Content, error) {
	// 等待任务完成
	task, err := c.WaitForCompletion(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.Data.FullZipURL == "" {
		return nil, fmt.Errorf("task completed but no zip url returned")
	}

	// 下载 ZIP 到临时目录
	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, taskID+".zip")
	defer os.Remove(zipPath) // 清理临时文件

	if err := result.DownloadZip(ctx, task.Data.FullZipURL, zipPath); err != nil {
		return nil, fmt.Errorf("download zip failed: %w", err)
	}

	// 提取内容
	content, err := result.ExtractAll(zipPath)
	if err != nil {
		return nil, fmt.Errorf("extract content failed: %w", err)
	}

	return content, nil
}

// DownloadResult 下载任务结果到指定路径（不解压）
func (c *Client) DownloadResult(ctx context.Context, taskID, destPath string) error {
	task, err := c.WaitForCompletion(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Data.FullZipURL == "" {
		return fmt.Errorf("task completed but no zip url returned")
	}

	return result.DownloadZip(ctx, task.Data.FullZipURL, destPath)
}
