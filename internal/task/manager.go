package task

import (
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
