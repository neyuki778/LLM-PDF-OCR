package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	llm "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM"
)

const defaultSubTaskTimeout = 8 * time.Minute

// 初始化worker pool
func NewWorkerPool(workerCount int, processor llm.PDFProcessor) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	// client, _ := genai.NewClient(ctx, nil)

	return &WorkerPool{
		workerCount: workerCount,
		taskQueue:   make(chan *SubTask, 100),
		resultChan:  make(chan *CompletionSignal, 10),
		processor:   processor,
		taskTimeout: defaultSubTaskTimeout,
		ctx:         ctx,
		cancel:      cancel,
		wg:          sync.WaitGroup{},
	}
}

// 启动worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// 向worker pool中放入任务

func (wp *WorkerPool) Submit(task *SubTask, timeout time.Duration) error {
	select {
	case wp.taskQueue <- task:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("Timeout")
	}
}

// 从worker pool的task queue中取任务
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for task := range wp.taskQueue {
		wp.processTask(task)
	}
}

// 关闭worker pool中的channels
func (wp *WorkerPool) Shutdown() {
	close(wp.taskQueue)
	wp.wg.Wait()
	close(wp.resultChan)
}

func (wp *WorkerPool) ResultChan() <-chan *CompletionSignal {
	return wp.resultChan
}

func (wp *WorkerPool) processTask(task *SubTask) {
	if task == nil {
		log.Printf("[worker] received nil subtask, skip")
		return
	}

	signal := &CompletionSignal{
		SubTaskID: task.ID,
		ParentID:  task.ParentID,
	}
	shouldEmit := false
	defer func() {
		if r := recover(); r != nil {
			shouldEmit = true
			signal.Success = false
			signal.Error = fmt.Errorf("panic while processing subtask: %v", r)
			log.Printf(
				"[worker] panic recovered parent_id=%s subtask_id=%s panic=%v\n%s",
				task.ParentID,
				task.ID,
				r,
				debug.Stack(),
			)
		}
		if shouldEmit {
			wp.resultChan <- signal
		}
	}()

	// 设置默认重试次数
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	taskCtx, cancel := context.WithTimeout(wp.ctx, wp.taskTimeout)
	defer cancel()

	var content string
	var err error
	for ; task.RetryCount < task.MaxRetries; task.RetryCount++ {
		if taskCtx.Err() != nil {
			err = taskCtx.Err()
			break
		}
		attempt := task.RetryCount + 1
		content, err = wp.processor.ProcessPDF(taskCtx, task.PDFPath)
		if err == nil {
			break
		}
		log.Printf(
			"[worker] subtask attempt failed parent_id=%s subtask_id=%s attempt=%d/%d err=%v",
			task.ParentID,
			task.ID,
			attempt,
			task.MaxRetries,
			err,
		)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			break
		}
		// 指数退避
		backoff := time.Duration(math.Pow(2, float64(task.RetryCount))) * time.Second
		if !sleepWithContext(taskCtx, backoff) {
			err = taskCtx.Err()
			break
		}
	}

	// 多次尝试失败
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("subtask timeout after %s: %w", wp.taskTimeout, err)
		}
		signal.Success = false
		signal.Error = err
		shouldEmit = true
		return
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(task.OutputPath), 0755); err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("failed to create directory: %w", err)
		shouldEmit = true
		return
	}

	// 写入文件
	file, err := os.OpenFile(task.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("can't open file %s: %w", task.OutputPath, err)
		shouldEmit = true
		return
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("failed to write content: %w", err)
		shouldEmit = true
		return
	}

	signal.Success = true
	signal.Error = nil
	shouldEmit = true
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// GetStatus 返回 WorkerPool 当前状态
func (wp *WorkerPool) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"worker_count":       wp.workerCount,
		"queue_length":       len(wp.taskQueue),
		"queue_capacity":     cap(wp.taskQueue),
		"result_chan_length": len(wp.resultChan),
	}
}
