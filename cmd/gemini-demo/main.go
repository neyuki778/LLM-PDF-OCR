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
	
	// 测试图片识别
	// imgpath := "test/imgs/test.png"
	// AnalyzeImageInline(ctx, client, imgpath)

	// 测试pdf转换
	// pdfpath := "test/pdfs/期末-2022编译原理期末卷.pdf"
	// AnalyzePDFInline(ctx, client, pdfpath)

	// 测试上传pdf再抓换
	// AnalyzePDFByUpload(ctx, client, pdfpath)

	// 测试流式传输
	StreamResponse(ctx, client)
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

func AnalyzePDFInline(ctx context.Context, client *genai.Client, filename string) {
    pdfBytes, err := os.ReadFile(filename)
    if err!= nil {
        log.Fatalf("读取文件失败: %v", err)
    }

	pdfPart := []*genai.Part{
		&genai.Part{
			InlineData: &genai.Blob{
				MIMEType: "application/pdf",
				Data: pdfBytes,
			},
		},
		genai.NewPartFromText("识别pdf并转换成合适的markdown格式"),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(pdfPart, genai.RoleUser),
	}

	result, _ := client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
        contents,
        nil,
	)

	fmt.Printf(result.Text())
}

func AnalyzePDFByUpload(ctx context.Context, client *genai.Client, filename string) {
	uploadConfig := &genai.UploadFileConfig{MIMEType: "application/pdf"}
	uploadedFile, _ := client.Files.UploadFromPath(ctx, filename, uploadConfig)
	fmt.Printf("PDF文件已上传, URL:%s", uploadedFile.URI)

	pdfPart := []*genai.Part{
		genai.NewPartFromURI(uploadedFile.URI, uploadedFile.MIMEType),
		genai.NewPartFromText("识别pdf并转换成合适的markdown格式"),
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(pdfPart, genai.RoleUser),
	}

	result, _ := client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
        contents,
        nil,
	)

	fmt.Printf(result.Text())
}

func StreamResponse(ctx context.Context, client *genai.Client) {
    iter := client.Models.GenerateContentStream(ctx, "gemini-2.5-flash", genai.Text("写一首关于Go语言的长诗。(我在做流式传输测试)"), nil)

	// Go 1.23+ 风格迭代
    // 如果使用旧版本Go，这里会有不同的写法，但SDK设计倾向于新标准
    for resp, err := range iter {
        if err!= nil {
            log.Printf("流传输错误: %v", err)
            break
        }
        // 实时打印每个分块（Chunk）的文本
        // fmt.Print(resp.Text()) 
		currText := resp.Text()

		for _, char := range currText {
			fmt.Print(string(char))
			time.Sleep(15 * time.Millisecond)
		}
    }
    fmt.Println() // 换行
}