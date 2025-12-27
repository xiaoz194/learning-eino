package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		Model:   "deepseek-chat",
		BaseURL: "https://api.deepseek.com",
	})
	if err != nil {
		log.Fatalf("创建失败: %v", err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的技术博主"),
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("开始对话（输入 'exit' 退出）：")

	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}

		userInput := scanner.Text()
		if userInput == "exit" {
			fmt.Println("再见！")
			break
		}

		if userInput == "" {
			continue
		}

		messages = append(messages, schema.UserMessage(userInput))
		// 流式生成
		var stream *schema.StreamReader[*schema.Message]

		stream, err = chatModel.Stream(ctx, messages)
		if err != nil {
			log.Fatalf("流式生成失败: %v", err)
		}
		fmt.Print("AI 回复: ")
		var respBuilder strings.Builder
		// 逐块接收并打印，同时收集完整响应
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// 流结束
					break
				}
				log.Fatalf("接收失败: %v", err)
			}

			// 打印内容（打字机效果）并收集
			fmt.Print(chunk.Content)
			respBuilder.WriteString(chunk.Content)
		}
		// 立即关闭流
		stream.Close()
		// 将完整的 assistant 回复追加到对话中（设置 Role 避免空值）
		// 如果不加这行，模型回复就会产生重复上一轮的回答，加上就正常了
		// Assistant is the role of an assistant, means the message is returned by ChatModel.
		// 正确的顺序应是 system、user、assistant、user。省略 assistant 会造成两个连续的 user，
		// 模型会尝试“补回答”之前未被记录的用户内容，导致重复或续写先前回复
		messages = append(messages, &schema.Message{Role: "assistant", Content: respBuilder.String()})
	}

}
