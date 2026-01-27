package worker

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	gemini "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM/gemini"
	"google.golang.org/genai"
)

// 初始化worker pool
func NewWorkerPool (workerCount int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := genai.NewClient(ctx, nil)

	return &WorkerPool{
		workerCount: workerCount,
		taskQueue: make(chan *SubTask, 100),
		resultChan: make(chan *CompletionSignal, 10),
		geminiClient: client,
		ctx: ctx,
		cancel: cancel,
		wg: sync.WaitGroup{},
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

func (wp *WorkerPool) Submit (task *SubTask, timeout time.Duration) error {
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
	// 设置默认重试次数
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	var content string
	var err error
	for ; task.RetryCount < task.MaxRetries; task.RetryCount++ {
		content, err = gemini.ProcessPDF(wp.ctx, wp.geminiClient, task.PDFPath)
		if err == nil {
			break
		}
		// 指数退避
		time.Sleep(time.Duration(math.Pow(2, float64(task.RetryCount))) * time.Second)
	}

	signal := &CompletionSignal{
		SubTaskID: task.ID,
		ParentID:  task.ParentID,
	}

	// 多次尝试失败
	if err != nil {
		signal.Success = false
		signal.Error = err
		wp.resultChan <- signal
		return
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(task.OutputPath), 0755); err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("failed to create directory: %w", err)
		wp.resultChan <- signal
		return
	}

	// 写入文件
	file, err := os.OpenFile(task.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("can't open file %s: %w", task.OutputPath, err)
		wp.resultChan <- signal
		return
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		signal.Success = false
		signal.Error = fmt.Errorf("failed to write content: %w", err)
		wp.resultChan <- signal
		return
	}

	signal.Success = true
	signal.Error = nil
	wp.resultChan <- signal
}