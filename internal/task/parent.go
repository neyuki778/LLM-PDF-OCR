package task

import (
	"fmt"
	"path/filepath"

	worker "github.com/neyuki778/LLM-PDF-OCR/internal/worker"
)

func NewParentTask (id, pdfPath, workDir string) *ParentTask {
	return &ParentTask{
		ID: id,
		OriginalPDF: pdfPath,
		WorkDir: workDir,
		OutputPath: filepath.Join(workDir, "result.md"),
		TotalShards: 0,
		SubTasks: make(map[string]*SubTaskMeta),
		CompletedCount: 0,
		FailedTasks: make([]string, 0),
		Status: StatusPending,
	}
}

func (pt *ParentTask) OnSubTaskComplete(signal *worker.CompletionSignal) error {
	if pt.ID != signal.ParentID {
		return fmt.Errorf("Submit a wrong subtask!")
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if signal.Success {
		pt.SubTasks[signal.SubTaskID].Status = SubTaskSuccess
		pt.CompletedCount++
	} else {
		pt.SubTasks[signal.SubTaskID].Status = SubTaskFailed
		pt.FailedTasks = append(pt.FailedTasks, signal.SubTaskID)
	}
	return nil
}

func (pt *ParentTask) IsAllDone() bool {
	return pt.CompletedCount == pt.TotalShards
}