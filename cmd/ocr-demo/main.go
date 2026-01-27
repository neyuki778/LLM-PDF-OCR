package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/neyuki778/LLM-PDF-OCR/internal/task"
)

func main() {
	godotenv.Load()

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <pdf_path>")
	}
	pdfPath := os.Args[1]

	tm := task.NewTaskManager(3)
	if err := tm.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
	defer tm.ShutDown()

	taskID, err := tm.CreateTask(pdfPath)
	if err != nil {
		log.Fatalf("Failed to create task: %v", err) 
	}
	fmt.Printf("Task created: %s\n", taskID) 
	
	if err := tm.SubmitTaskToPool(taskID, 10*time.Second); err != nil {
		log.Fatalf("Failed to submit: %v", err)
	}
	fmt.Println("Processing...") 
	if err := tm.WaitForTask(taskID, 5*time.Minute); err != nil {
		log.Fatalf("Failed while waiting: %v", err)
	}

	fmt.Printf("Done! Output: ./output/%s/result.md\n", taskID)
}
