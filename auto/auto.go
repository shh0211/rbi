package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// 远程调试端点 URL （remote-server-ip:9333）
	remoteURL := "ws://202.63.172.204:9444"

	// 创建上下文，连接到远程 Chrome 实例
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), remoteURL)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 执行浏览器操作
	var pageTitle string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.baidu.com"),
		chromedp.Title(&pageTitle),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Page title: %s\n", pageTitle)
}
