package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	} else {
		pt.SubTasks[signal.SubTaskID].Status = SubTaskFailed
		pt.FailedTasks = append(pt.FailedTasks, signal.SubTaskID)
	}
	pt.CompletedCount++
	return nil
}

func (pt *ParentTask) IsAllDone() bool {
	return pt.CompletedCount == pt.TotalShards
}

func (pt *ParentTask) doAggregate() error {
	file, err := os.OpenFile(pt.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, subTaskMeta := range pt.SortSubTasksByPageStart() {
		if subTaskMeta.Status == SubTaskSuccess {
			content, err := os.ReadFile(subTaskMeta.TempFilePath)
			if err != nil {
				return err
			}
			_, err = file.Write(content)
            if err != nil {
            	return err
            }
		} else {
			_, err = fmt.Fprintf(file, "<!-- [OCR Failed] Pages %d-%d: %s -->\n", 
            	subTaskMeta.PageStart, subTaskMeta.PageEnd, subTaskMeta.ID)
			if err != nil {
				return err
			}
		}
	}

	// 清除临时文件
	for _, subTask := range pt.SubTasks {
		os.Remove(subTask.SplitPDFPath)
		os.Remove(subTask.TempFilePath)
	}

	// 更新parent task status
	pt.Status = StatusCompleted

	return nil
}

func (pt *ParentTask) SortSubTasksByPageStart() []*SubTaskMeta {
	list := make([]*SubTaskMeta, 0, len(pt.SubTasks))
	for _, subTask := range pt.SubTasks {
		list = append(list, subTask)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].PageStart < list[j].PageStart
	})

	return list
}

func (pt *ParentTask) Aggregate() error {
	var err error
	pt.aggregateOnce.Do(func() {
		err = pt.doAggregate()
	})

	return err
}