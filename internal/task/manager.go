package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	worker "github.com/neyuki778/LLM-PDF-OCR/internal/worker"
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

func NewTaskManager(workCount int) *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*ParentTask),
		pool: worker.NewWorkerPool(workCount),
		stopChan: make(chan struct{}),
	}
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
			return fmt.Errorf("Wrong: %e", err)
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
		// to do
	}
	return nil
}

// 完整的任务创建功能, 包含pdf切分
func (tm *TaskManager) CreatTask(pdfPath string) (taskID string, err error) {
	taskID = uuid.New().String()
	workDir := filepath.Join("./output/", taskID)

	// 提取原文件名字
	baseName := filepath.Base(pdfPath) // "report.pdf"
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))  // "report"

	totalPages, err := pdf.GetPageCount(pdfPath)
	if err != nil {
		return taskID, err
	}

	// 目前固定每个PDF最大为5页
	span := 5
	totalShards := (totalPages + span - 1) / span

	parentTask := NewParentTask(taskID, pdfPath, workDir)
	parentTask.TotalShards = totalShards

	// 创建并填充sub-task
	for i := range totalShards {

		subTaskID := fmt.Sprintf("%s_%d", taskID, i+1)

		splitPath := filepath.Join(workDir, fmt.Sprintf("%s_%d.pdf", nameWithoutExt, i+1))
		tempFilePath := filepath.Join(workDir, fmt.Sprintf("page_%d.md", i+1))

		pageStart := i * span + 1
		pageEnd := min((i + 1) * span, totalPages)

		meta := SubTaskMeta{
			ID: subTaskID,
			PageStart: pageStart,
			PageEnd: pageEnd,
			SplitPDFPath: splitPath,
			TempFilePath: tempFilePath,
			Status: SubTaskPending,
			Error: nil,
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
