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
	//shh的本地浏览器调试
	remoteURL := "ws://202.63.172.204:9444"
	// 创建上下文，连接到远程 Chrome 实例
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), remoteURL)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// 设置登录界面的url
	loginUrl := "http://git.avocado.wiki"
	username := "sunhaohang@buaa.edu.cn"
	password := "St111535"
	// 执行浏览器操作
	var pageTitle string
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginUrl),
		chromedp.Click(`a.item svg.octicon-sign-in`, chromedp.ByQuery),
		chromedp.WaitVisible(`#user_name`, chromedp.ByID),
		chromedp.SendKeys(`#user_name`, username, chromedp.ByID),
		chromedp.WaitVisible(`#password`, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`button.ui.primary.button`, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Page title: %s\n", pageTitle)
	time.Sleep(30 * time.Second)
}
