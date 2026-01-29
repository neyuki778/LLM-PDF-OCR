package task

import (
	"sync"
)

// SubTaskMeta 子任务元信息（ParentTask用于追踪）
type SubTaskMeta struct {
	ID           string // 子任务ID
	PageStart    int    // 起始页码
	PageEnd      int    // 结束页码
	SplitPDFPath string // 分片PDF路径：./output/{parentID}/split_1.pdf
	TempFilePath string // 临时MD路径：./output/{parentID}/page_1.md
	Status       string // pending/processing/success/failed
	Error        error  // 失败时的错误信息
}

// ParentTask 父任务（对应一个完整的PDF处理请求）
type ParentTask struct {
	ID          string // 任务唯一ID（UUID）
	OriginalPDF string // 原始PDF路径（输入）
	WorkDir     string // 工作目录：./output/{ID}/
	OutputPath  string // 最终结果路径：./output/{ID}/result.md

	// 分片信息
	TotalShards int                     // 总分片数
	SubTasks    map[string]*SubTaskMeta // key: SubTaskID

	// 进度追踪
	CompletedCount int      // 已完成数量（成功+失败）
	FailedTasks    []string // 失败的SubTaskID列表

	// 状态
	Status string // pending/processing/completed/failed

	// 并发控制
	mu            sync.Mutex // 保护内部状态
	aggregateOnce sync.Once  // 保证Aggregate只执行一次
}

// TaskStatus 定义任务状态常量
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// SubTaskStatus 定义子任务状态常量
const (
	SubTaskPending    = "pending"
	SubTaskProcessing = "processing"
	SubTaskSuccess    = "success"
	SubTaskFailed     = "failed"
)
