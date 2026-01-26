package worker

import (
	"fmt"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	// 1. 创建Pool（3个worker用于测试）
	pool := NewWorkerPool(3)

	// 2. 启动Pool
	pool.Start()
	fmt.Println("Worker Pool 已启动")

	// 3. 启动结果监听器
	go func() {
		for signal := range pool.resultChan {
			fmt.Printf("收到结果: 任务%s, 成功=%v\n",
				signal.SubTaskID, signal.Success)
		}
	}()

	// 4. 提交5个测试任务
	for i := 1; i <= 5; i++ {
		task := &SubTask{
			ID:        fmt.Sprintf("task_%d", i),
			ParentID:  "parent_test",
			PDFPath:   "/fake/path.pdf",
			PageStart: i * 5,
			PageEnd:   i*5 + 4,
			MaxRetries: 3,
		}

		if err := pool.Submit(task, 2*time.Second); err != nil {
			t.Fatalf("提交任务失败: %v", err)
		}
		fmt.Printf("已提交任务: %s\n", task.ID)
	}

	// 5. 等待一段时间让任务处理完
	time.Sleep(2 * time.Second)

	// 6. 关闭Pool
	pool.Shutdown()
	fmt.Println("Worker Pool 已关闭")
}
