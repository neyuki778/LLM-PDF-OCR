package worker

import (
	"context"
	"sync"

	"google.golang.org/genai"
)

type SubTask struct {
	ID         string // 分片唯一ID                                                                                   
	ParentID   string // 所属父任务ID                                                                                 
	PDFPath    string // 分片PDF文件路径                                                                              
	PageStart  int    // 起始页码                                                                                     
	PageEnd    int    // 结束页码                                                                                     
	RetryCount int    // 当前重试次数                                                                                 
	MaxRetries int    // 最大重试次数（默认3）
}

type CompletionSignal struct {
	SubTaskID string // 分片ID                                                                                        
	ParentID  string // 父任务ID                                                                                      
	Success   bool   // 是否成功                                                                                      
	Error     error  // 失败时的错误信息 
}

type WorkerPool struct {
	workerCount  int                      // worker数量（固定5）                                                      
	taskQueue    chan *SubTask            // 任务队列（容量100）                                                      
	resultChan   chan *CompletionSignal   // 结果通道（容量10）                                                       
	geminiClient *genai.Client            // Gemini API客户端                                                         
	ctx          context.Context          // 上下文                                                                   
	cancel       context.CancelFunc       // 取消函数                                                                 
	wg           sync.WaitGroup           // 等待所有worker退出  
}