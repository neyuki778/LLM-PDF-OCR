package main

import (
	"archive/zip"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	mineru "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM/MinerU"
	result "github.com/neyuki778/LLM-PDF-OCR/pkg/result"
)

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	baseURL := getenv("MINERU_BASE_URL", "https://mineru.net")
	token := os.Getenv("MINERU_TOKEN")
	fileURL := os.Getenv("MINERU_FILE_URL")
	modelVersion := getenv("MINERU_MODEL_VERSION", "vlm")

	if len(os.Args) > 1 {
		fileURL = os.Args[1]
	}

	if token == "" || fileURL == "" {
		log.Fatalf("missing input: MINERU_TOKEN=%t FILE_URL=%t (env MINERU_FILE_URL or argv)", token != "", fileURL != "")
	}

	client := mineru.NewClient(baseURL, token, baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.CreateTask(ctx, mineru.CreateTaskRequest{
		URL:          fileURL,
		ModelVersion: modelVersion,
	})
	if err != nil {
		log.Fatalf("create task failed: %v", err)
	}

	fmt.Printf("Task created: %s\n", resp.Data.TaskID)
	fmt.Println("Waiting for completion and downloading zip...")

	outDir := filepath.Join("output", "mineru-demo")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("create output dir failed: %v", err)
	}

	zipPath := filepath.Join(outDir, resp.Data.TaskID+".zip")
	if err := client.DownloadResult(ctx, resp.Data.TaskID, zipPath); err != nil {
		log.Fatalf("download result failed: %v", err)
	}
	fmt.Printf("Zip saved: %s\n", zipPath)

	fmt.Println("\n=== Zip Entries ===")
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Fatalf("open zip failed: %v", err)
	}
	for _, f := range r.File {
		fmt.Printf("- %s (%d bytes)\n", f.Name, f.UncompressedSize64)
	}
	r.Close()

	extractDir := filepath.Join(outDir, resp.Data.TaskID)
	if err := result.ExtractToDir(zipPath, extractDir); err != nil {
		log.Fatalf("extract zip failed: %v", err)
	}
	fmt.Printf("\nExtracted to: %s\n", extractDir)
}


// https://pdf.kana.engineer/uploads/4317daf6-84bc-4568-b5ee-76efd9ad371d.pdf