package main

import (
	"context"
	"fmt"
	"log"
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
	
	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
        genai.Text("Explain how AI works in a few words"),
        nil,
	)

	if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Text())
}
