package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	FileURL    = "fileUrl"
	ByteLen    = 16
	UserId     = "uid"
	Connection = "Connection"
	Upgrade    = "Upgrade"
	// 定义端口范围
	minPort   = 10000
	maxPort   = 65535
	rangeSize = 10
)

type StartRequest struct {
	UserId  string `json:"userId"`
	FileUrl string `json:"fileUrl"`
}

type StopRequest struct {
	ContainerID string `json:"containerId"`
}

func randUid(len int) string {
	randomBytes := make([]byte, len)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println("Failed to generate random UID:", err)
		return ""
	}
	return hex.EncodeToString(randomBytes)
}

func getUid(r *http.Request) string {
	c, err := r.Cookie(UserId)
	if err != nil {
		fmt.Println("Error retrieving UID from cookie:", err)
		return ""
	}
	return c.Value
}

func startContainer(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	hasUid := getUid(r)
	if hasUid != "" && len(hasUid) == ByteLen*2 { // ByteLen*2 for hex representation
		req.UserId = hasUid
	} else {
		req.UserId = randUid(ByteLen)
	}

	queryParams := r.URL.Query()
	fileUrl := queryParams.Get(FileURL)
	if fileUrl == "" {
		http.Error(w, "Missing 'fileUrl' parameter", http.StatusBadRequest)
		return
	}
	req.FileUrl = fileUrl

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Failed to create Docker client", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	startPort, endPort, err := generateRandomPortRange(minPort, maxPort, rangeSize)
	if err != nil {
		// if range fail re open it
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	ok, err := checkPortRangeConflict(Db, startPort, rangeSize)
	if err != nil {
		// if range fail re open it
		fmt.Println("Check port fail:", err.Error())
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	if !ok {
		fmt.Println("Port conflict,So redirect")
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	env := []string{
		"NEKO_SCREEN=1920x1080@60",
		"NEKO_PASSWORD=ribZXCxcZXcXZCxzZ",
		"NEKO_PASSWORD_ADMIN=rbi",
		fmt.Sprintf("NEKO_EPR=%d-%d", startPort, endPort),
		"NEKO_NAT1TO1=202.63.172.204",
	}

	portBindings := map[nat.Port][]nat.PortBinding{}

	for i := startPort; i <= endPort; i++ {
		port := fmt.Sprintf("%d/udp", i)
		portBindings[nat.Port(port)] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", i)},
		}
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "wps",
		Env:   env,
	}, &container.HostConfig{
		ShmSize:      2 * 1024 * 1024 * 1024,
		PortBindings: portBindings,
		CapAdd:       strslice.StrSlice{"SYS_ADMIN"},
		AutoRemove:   true,
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: "./dist",
				Target: "/var/www",
			},
		},
	}, nil, nil, "neko_user_"+req.UserId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Docker container: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start Docker container %s", err.Error()), http.StatusInternalServerError)
		return
	}
	// 获取容器详细信息
	containerJSON, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		fmt.Println("Failed to inspect container:", err)
		return
	}

	// 获取容器的IP地址
	containerIP := containerJSON.NetworkSettings.IPAddress
	if containerIP == "" {
		fmt.Println("Container IP address not found")
	} else {
		fmt.Printf("Container IP address: %s\n", containerIP)
	}
	if containerIP == "" {
		fmt.Println("Container IP is not found")
		return
	}
	containerInfo := &ContainerInfo{
		ContainerId: resp.ID,
		MinPort:     startPort,
		IP:          containerIP,
	}
	err = Db.Save(containerInfo).Error
	if err != nil {
		fmt.Println("Save ContainerInfo err:", err)
	}
	cmd := fmt.Sprintf("url=%s && filename=$(basename $url) && wget -O \"$filename\" $url && wps \"$filename\"", req.FileUrl)
	execConfig := container.ExecOptions{
		Cmd:          strslice.StrSlice{"bash", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}

	time.Sleep(2 * time.Second)
	execIDResp, err := cli.ContainerExecCreate(ctx, resp.ID, execConfig)
	if err != nil {
		http.Error(w, "Failed to create exec instance", http.StatusInternalServerError)
		return
	}

	if err := cli.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{}); err != nil {
		http.Error(w, "Failed to start exec instance", http.StatusInternalServerError)
		return
	}

	fmt.Println("Command executed inside container")
	if resp.ID != "" {
		url := fmt.Sprintf("/%s", resp.ID)
		http.Redirect(w, r, url, http.StatusFound)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to start the service"))
	}
}

func stopContainer(w http.ResponseWriter, r *http.Request) {
	var req StopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Failed to create Docker client", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	if err := cli.ContainerStop(ctx, req.ContainerID, container.StopOptions{}); err != nil {
		http.Error(w, "Failed to stop Docker container", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Container stopped and removed successfully"))
}

// Define an upgrader to upgrade HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust this function if needed for origin checks
	},
}

func dynamicProxy(w http.ResponseWriter, r *http.Request) {
	// 假设路径格式为 /{container_id}/{path_to_proxy}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	containerID := parts[1]
	ip, err := getContainerIP(containerID)
	if err != nil {
		http.Error(w, "Failed to get container port", http.StatusInternalServerError)
		return
	}

	if ip == "" {
		http.Error(w, "Get Container IP from Db is null", http.StatusInternalServerError)
		return
	}
	target := fmt.Sprintf("http://%s:8080", ip)
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Failed to parse target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		pathToProxy := "/" + strings.Join(parts[2:], "/")

		fmt.Println(target)
		req.Header = r.Header
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		if len(containerID) == len("0f27e486215643f62403a9f7d97a620b12f24667a93131db3838aff6f520c0de") {
			req.URL.Path = pathToProxy
		}

		req.Host = targetURL.Host
	}
	if IsWebsocketRequest(r) {
		revproxy := httputil.ReverseProxy{
			Director: proxy.Director,
		}
		revproxy.ServeHTTP(w, r)
	} else {
		proxy.ServeHTTP(w, r)
	}
}

func getContainerIP(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	dbRes := &ContainerInfo{}
	err := Db.Find(dbRes, &ContainerInfo{ContainerId: containerID}).Error
	if err != nil {
		return "", err
	}
	return dbRes.ContainerId, nil
}
func getContainerPort(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	return "8080", nil
}

type ContainerInfo struct {
	ID          int64 `gorm:"primaryKey"`
	ContainerId string
	IP          string
	Port        string
	MinPort     int
}

// 检查新生成的端口范围是否与数据库中的记录冲突
func checkPortRangeConflict(db *gorm.DB, newMinPort, rangeSize int) (bool, error) {
	var conflicts int64
	newMaxPort := newMinPort + rangeSize - 1

	err := db.Model(&ContainerInfo{}).Where("? BETWEEN MinPort AND MinPort + ? - 1 OR ? BETWEEN MinPort AND MinPort + ? - 1",
		newMinPort, rangeSize, newMaxPort, rangeSize).Count(&conflicts).Error

	if err != nil {
		return false, err
	}

	return conflicts == 0, nil
}

func initDB() (*gorm.DB, error) {
	// 初始化 SQLite 数据库连接
	db, err := gorm.Open(sqlite.Open("rbi.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// 自动迁移模式，创建表
	db.AutoMigrate(&ContainerInfo{})

	// 创建一个新的用户
	return db, err
}

var Db *gorm.DB

func main() {
	db, err := initDB()
	if err != nil {
		panic(err.Error())
	}
	Db = db
	router := mux.NewRouter()
	router.HandleFunc("/start", startContainer).Methods(http.MethodGet)
	router.HandleFunc("/stop", stopContainer).Methods(http.MethodPost)
	router.PathPrefix("/").HandlerFunc(dynamicProxy)

	fmt.Println("Starting server on port 18083")
	if err := http.ListenAndServe(":18083", router); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

func IsWebsocketRequest(req *http.Request) bool {
	containsHeader := func(name, value string) bool {
		items := strings.Split(req.Header.Get(name), ",")
		for _, item := range items {
			if value == strings.ToLower(strings.TrimSpace(item)) {
				return true
			}
		}
		return false
	}
	return containsHeader(Connection, "upgrade") && containsHeader(Upgrade, "websocket")
}

func findAvailablePort() (int, error) {
	// 使用 net.Listen 绑定到端口 0，操作系统会自动选择一个可用的端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// 获取监听地址中的端口号
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// 生成随机端口范围
func generateRandomPortRange(minPort, maxPort, rangeSize int) (int, int, error) {
	rand.Seed(time.Now().UnixNano())
	startPort := rand.Intn(maxPort-minPort-rangeSize+1) + minPort
	endPort := startPort + rangeSize - 1

	// 检查端口范围是否可用
	for port := startPort; port <= endPort; port++ {
		if !isPortAvailable("udp", port) {
			return 0, 0, fmt.Errorf("port %d is not available", port)
		}
	}

	return startPort, endPort, nil
}

// 检查端口是否可用
func isPortAvailable(network string, port int) bool {
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}
