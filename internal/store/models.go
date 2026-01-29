package store

import "time"

type TaskRecord struct {
    ID         string    `json:"id"`
    Status     string    `json:"status"`      // pending, processing, completed, failed
    PDFPath    string    `json:"pdf_path"`    // 原始 PDF 路径
    ResultPath string    `json:"result_path"` // 结果 Markdown 路径
    TotalPages int       `json:"total_pages"`
    Error      string    `json:"error,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
