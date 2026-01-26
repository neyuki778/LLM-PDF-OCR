package pdf

import (
	"context"
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func SplitPDF(ctx context.Context, inputPath, outputDir string, span int) error {
	if inputPath == "" || outputDir == "" {
		return fmt.Errorf("Input/OutputPath should not be empty!")
	} else if span <= 0 {
		return  fmt.Errorf("Span should bigger than 0!")	
	}

	// 创建输出目录
	_ = os.MkdirAll(outputDir, 0755)
	
	err := api.SplitFile(inputPath, outputDir, span, nil)
	if err != nil {
		fmt.Printf("拆分失败: %v\n", err)
		return err
	}
	return err
}

func GetPageCount (pdfPath string) (int, error) {
	return api.PageCountFile(pdfPath)
}