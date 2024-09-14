package main

import (
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"rbi/config"
	"rbi/containers"
	"rbi/proxy"
	"rbi/user"
)

func main() {
	// 读取配置文件
	config.ReadConfig("config.yml")
	// 初始化 TTL 检查
	containers.InitTTLCheck()
	// 设置路由
	router := mux.NewRouter()
	containers.RegisterRoutes(router)
	proxy.RegisterRoutes(router)
	user.RegisterRoutes(router)
	// 启动服务
	fmt.Println("Starting server on port 18083")
	if err := http.ListenAndServe(":18083", router); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
