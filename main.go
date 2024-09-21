package main

import (
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"rbi/automation"
	"rbi/config"
	"rbi/containers"
	"rbi/middleware"
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
	// 使用 CORS 中间件
	router.Use(middleware.CORS)
	// 注册
	containers.RegisterRoutes(router)
	user.RegisterRoutes(router)
	automation.RegisterRoutes(router)
	proxy.RegisterRoutes(router)
	// 启动服务
	fmt.Println("Starting server on port 18083")
	if err := http.ListenAndServe(":18083", router); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
