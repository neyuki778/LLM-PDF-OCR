package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/genai"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background() ,30 * time.Second)
	defer cancel()

	// 从环境变量中读取api_key
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	
	// modelName := "gemini-3-flash-preview"
	// result, err := client.Models.GenerateContent(
	// 	ctx,
	// 	modelName,
    //     genai.Text("Explain how AI works in a few words"),
    //     nil,
	// )

	// if err != nil {
    //     log.Fatal(err)
    // }
    // fmt.Println(result.Text())
	filepath := "test/imgs/test.png"
	AnalyzeImageInline(ctx, client, filepath)
}

func AnalyzeImageInline(ctx context.Context, client *genai.Client, filename string) {
    // 1. 读取本地图片文件
    imgBytes, err := os.ReadFile(filename)
    if err!= nil {
        log.Fatalf("读取文件失败: %v", err)
    }

    // 2. 构建Blob Part
    // 注意：MIMEType必须准确，如 image/png, image/jpeg, image/webp
    imgPart := &genai.Blob{
        MIMEType: "image/png",
        Data:     imgBytes,
    }

    // 3. 构建文本提示 Part
    textPart := "精简描述这张图片中的内容"

    // 4. 组合请求内容
    // 顺序很重要：通常先图后文，或图文穿插
    content := &genai.Content{
        Parts: []*genai.Part{
			{InlineData: imgPart},
			{Text: textPart},
		},
    }

    // 5. 调用模型
    resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{content}, nil)
    if err!= nil {
        log.Fatal(err)
    }
    
    fmt.Println(resp.Text())
}