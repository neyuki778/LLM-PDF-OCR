package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

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
		fmt.Printf("处理任务: %s, 页码: %d-%d\n",                                                                       
              task.ID, task.PageStart, task.PageEnd)
		
		// 处理成功
		wp.resultChan <- &CompletionSignal{
			SubTaskID: task.ID,
			ParentID: task.ParentID,
			Success: true,
			Error: nil,
		}
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