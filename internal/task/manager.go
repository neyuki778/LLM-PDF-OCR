package task

import (
	"fmt"
	"sync"

	worker "github.com/neyuki778/LLM-PDF-OCR/internal/worker"
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

func NewTaskManaegr(workCount int) *TaskManager {
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
			tm.handleResult(signal)
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