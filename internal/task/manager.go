package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	worker "github.com/neyuki778/LLM-PDF-OCR/internal/worker"
	llm "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM"
	pdf "github.com/neyuki778/LLM-PDF-OCR/pkg/pdf"
)

// TaskManager 任务管理器（全局单例）
type TaskManager struct {
	// 任务存储
	tasks map[string]*ParentTask // key: ParentTask.ID
	mu    sync.RWMutex           // 保护tasks map

	// Worker Pool
	pool *worker.WorkerPool

	// 生命周期控制
	stopChan chan struct{} // 用于停止监听器
}

func NewTaskManager(workCount int, config llm.Config) (*TaskManager, error) {
	processor, err := llm.NewProcessor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create processor: %w", err)
	}
	return &TaskManager{
		tasks:    make(map[string]*ParentTask),
		pool:     worker.NewWorkerPool(workCount, processor),
		stopChan: make(chan struct{}),
	}, nil
}

func (tm *TaskManager) Start() error {
	tm.pool.Start()
	go tm.ListenResult()
	return nil
}

func (tm *TaskManager) ListenResult() error {
	for {
		select {
		case signal, ok := <-tm.pool.ResultChan():
			if !ok {
				return fmt.Errorf("Something wrong with ResultChan!")
			}
			err := tm.handleResult(signal)
			if err != nil {
				return fmt.Errorf("Wrong: %e", err)
			}
		case <-tm.stopChan:
			return nil
		}
	}
}

func (tm *TaskManager) ShutDown() error {
	close(tm.stopChan)
	tm.pool.Shutdown()
	return nil
}

func (tm *TaskManager) handleResult(signal *worker.CompletionSignal) error {
	parentTask, exists := tm.tasks[signal.ParentID]
	if !exists {
		return fmt.Errorf("%s don't exists!", signal.ParentID)
	}

	if err := parentTask.OnSubTaskComplete(signal); err != nil {
		return err
	}

	if parentTask.IsAllDone() {
		go parentTask.Aggregate()
	}
	return nil
}

// 完整的任务创建功能, 包含pdf切分
func (tm *TaskManager) CreateTask(pdfPath string) (taskID string, err error) {
	taskID = uuid.New().String()
	workDir := filepath.Join("./output/", taskID)

	// 提取原文件名字
	baseName := filepath.Base(pdfPath)                                     // "report.pdf"
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName)) // "report"

	totalPages, err := pdf.GetPageCount(pdfPath)
	if err != nil {
		return taskID, err
	}
	maxPageCount := 30
	if totalPages >= maxPageCount {
		return "", fmt.Errorf("Total pages: %d should less than %d", totalPages, maxPageCount)
	}

	// 目前固定每个PDF最大2页
	span := 2
	totalShards := (totalPages + span - 1) / span

	parentTask := NewParentTask(taskID, pdfPath, workDir)
	parentTask.TotalShards = totalShards

	// 创建并填充sub-task
	for i := range totalShards {

		subTaskID := fmt.Sprintf("%s_%d", taskID, i+1)

		pageStart := i*span + 1
		pageEnd := min((i+1)*span, totalPages)

		splitFileName := fmt.Sprintf("%s_%d-%d.pdf", nameWithoutExt, pageStart, pageEnd)
		if pageStart == pageEnd {
			splitFileName = fmt.Sprintf("%s_%d.pdf", nameWithoutExt, pageStart)
		}
		splitPath := filepath.Join(workDir, splitFileName)
		tempFilePath := filepath.Join(workDir, fmt.Sprintf("page_%d.md", i+1))

		meta := SubTaskMeta{
			ID:           subTaskID,
			PageStart:    pageStart,
			PageEnd:      pageEnd,
			SplitPDFPath: splitPath,
			TempFilePath: tempFilePath,
			Status:       SubTaskPending,
			Error:        nil,
		}

		parentTask.SubTasks[subTaskID] = &meta
	}

	// 切分pdf
	ctx := context.Background()
	if err := pdf.SplitPDF(ctx, pdfPath, workDir, span); err != nil {
		return "", fmt.Errorf("failed to split PDF: %w", err)
	}

	tm.mu.Lock()
	tm.tasks[taskID] = parentTask
	tm.mu.Unlock()

	return taskID, nil
}

func (tm *TaskManager) SubmitTaskToPool(taskID string, timeout time.Duration) error {
	tm.mu.RLock()
	parentTask, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	for _, subTask := range parentTask.SubTasks {
		workerTask := &worker.SubTask{
			ID:         subTask.ID,
			ParentID:   taskID,
			PDFPath:    subTask.SplitPDFPath,
			OutputPath: subTask.TempFilePath,
			PageStart:  subTask.PageStart,
			PageEnd:    subTask.PageEnd,
		}

		if err := tm.pool.Submit(workerTask, timeout); err != nil {
			return fmt.Errorf("failed to submit subtask %s: %w", subTask.ID, err)
		}
	}

	parentTask.Status = StatusProcessing
	return nil
}

func (tm *TaskManager) WaitForTask(taskID string, timeout time.Duration) error {
	tm.mu.RLock()
	parentTask, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	ddl := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ddl:
			return fmt.Errorf("timeout waiting for task %s", taskID)
		case <-ticker.C:
			if parentTask.Status == StatusCompleted {
				return nil
			}
		}
	}

}

func (tm *TaskManager) GetTask(taskID string) *ParentTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tasks[taskID]
}

// GetStatus 返回 TaskManager 的整体状态信息
func (tm *TaskManager) GetStatus() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 统计任务状态
	statusCount := map[string]int{
		"pending":    0,
		"processing": 0,
		"completed":  0,
	}

	for _, task := range tm.tasks {
		statusCount[task.Status]++
	}

	return map[string]interface{}{
		"total_tasks": len(tm.tasks),
		"task_status": statusCount,
		"worker_pool": tm.pool.GetStatus(),
	}
}
