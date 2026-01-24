package result

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractMarkdown 从 ZIP 文件中提取 full.md 的内容
func ExtractMarkdown(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("open zip failed: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == "full.md" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open full.md failed: %w", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("read full.md failed: %w", err)
			}

			return string(content), nil
		}
	}

	return "", fmt.Errorf("full.md not found in zip")
}

// ExtractAll 提取 ZIP 中的所有关键文件内容
func ExtractAll(zipPath string) (*Content, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip failed: %w", err)
	}
	defer r.Close()

	result := &Content{}
	foundMarkdown := false

	for _, f := range r.File {
		basename := filepath.Base(f.Name)

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s failed: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s failed: %w", f.Name, err)
		}

		switch basename {
		case "full.md":
			result.Markdown = string(content)
			foundMarkdown = true
		case "layout.json":
			result.LayoutJSON = string(content)
		default:
			// 检查是否是 PDF 文件（源文件）
			if strings.HasSuffix(strings.ToLower(basename), ".pdf") {
				result.SourcePDF = content
			}
		}
	}

	if !foundMarkdown {
		return nil, fmt.Errorf("full.md not found in zip")
	}

	return result, nil
}

// ExtractToDir 将 ZIP 文件解压到指定目录
func ExtractToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip failed: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create destination dir failed: %w", err)
	}

	for _, f := range r.File {
		destPath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return fmt.Errorf("create dir %s failed: %w", destPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("create parent dir failed: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s failed: %w", f.Name, err)
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s failed: %w", destPath, err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("extract %s failed: %w", f.Name, err)
		}
	}

	return nil
}
