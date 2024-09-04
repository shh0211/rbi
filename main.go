package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Config struct {
	TTLMinutes              int `yaml:"ttlMinutes"`
	CheckIntervalSeconds    int `yaml:"checkIntervalSeconds"`
	wsUpdateIntervalSeconds int `yaml:"wsUpdateIntervalSeconds"`
}

const (
	FileURL    = "fileUrl"
	ByteLen    = 16
	UserId     = "uid"
	Connection = "Connection"
	Upgrade    = "Upgrade"
	// 定义端口范围
	minPort   = 10000
	maxPort   = 65535
	rangeSize = 100
)

var MapInfo Mapping

type Mapping struct {
	M map[string]string
	sync.RWMutex
}

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
		fmt.Println("port err:", err.Error())
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
		"NEKO_PASSWORD=rbi",
		"NEKO_PASSWORD_ADMIN=rbi",
		fmt.Sprintf("NEKO_EPR=%d-%d", startPort, endPort),
		"NEKO_NAT1TO1=202.63.172.204",
	}

	portBindings := make(nat.PortMap, 0)
	exposedPorts := make(nat.PortSet, 0)
	for i := startPort; i <= endPort; i++ {
		port := fmt.Sprintf("%d/udp", i)
		p, _ := nat.NewPort("udp", fmt.Sprintf("%d", i))
		exposedPorts[p] = struct{}{} // 设置容器的暴露端口
		portBindings[nat.Port(port)] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", i)},
		}
	}

	//cwd, err := os.Getwd()
	//if err != nil {
	//	fmt.Println("Error getting current working directory:", err)
	//	return
	//}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        "wps",
		ExposedPorts: exposedPorts,
		Env:          env,
	}, &container.HostConfig{
		ShmSize:      2 * 1024 * 1024 * 1024,
		PortBindings: portBindings,
		CapAdd:       strslice.StrSlice{"SYS_ADMIN"},
		AutoRemove:   true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/opt/neko/dist",
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
		ExpireAt:    time.Now().Add(time.Duration(ttl) * time.Minute),
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
		url := fmt.Sprintf("/%s/", resp.ID)
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

	tx := Db.Begin()
	if tx.Error != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return

	}
	ctx := context.Background()

	if err := cli.ContainerStop(ctx, req.ContainerID, container.StopOptions{}); err != nil {
		tx.Rollback()
		http.Error(w, "Failed to stop Docker container", http.StatusInternalServerError)
		return
	}
	// 删除数据库中对应的记录
	if err := tx.Where("container_id = ?", req.ContainerID).Delete(&ContainerInfo{}).Error; err != nil {
		tx.Rollback() // 如果删除记录失败，则回滚事务
		http.Error(w, "Failed to delete container record", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Database transaction commit failed", http.StatusInternalServerError)
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

// 更新容器ttl
func updateContainerTTL(containerID string) {
	result := Db.Model(&ContainerInfo{}).Where("container_id = ?", containerID).Update("expire_at", time.Now().Add(time.Duration(ttl)*time.Minute))
	if result.Error != nil {
		log.Printf("Failed to update container TTL: %v", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("Container %s not found", containerID)
		return
	}
	log.Printf("Container %s TTL updated successfully", containerID)
	return
}

// 检查并删除过期容器
func checkAndDeleteExpiredContainers() {
	var expiredContainers []ContainerInfo
	// 从数据库中查询所有已经过期的容器
	result := Db.Where("expire_at <= ?", time.Now()).Find(&expiredContainers)
	if result.Error != nil {
		log.Printf("Failed to fetch expired containers: %v", result.Error)
		return
	}

	for _, container := range expiredContainers {
		if err := deleteDockerContainer(container.ContainerId); err != nil {
			log.Printf("Error handling container %s: %v", container.ContainerId, err)
		} else {
			// 从数据库中删除容器记录
			Db.Delete(&container)
		}
	}
}
func deleteDockerContainer(containerID string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	// 停止容器
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		log.Printf("Failed to stop container %s: %v", containerID, err)
		return err
	}

	fmt.Printf("Container %s removed successfully\n", containerID)
	return nil
}
func dynamicProxy(w http.ResponseWriter, r *http.Request) {
	// 假设路径格式为 /{container_id}/{path_to_proxy}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	containerID := parts[1]

	var ip string
	if v, ok := MapInfo.M[containerID]; ok {
		ip = v
	} else {
		var err error
		ip, err = getContainerIP(containerID)
		if err != nil {
			http.Error(w, "Failed to get container port", http.StatusInternalServerError)
			return
		}
		MapInfo.RWMutex.Lock()
		MapInfo.M[containerID] = ip
		MapInfo.RWMutex.Unlock()
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
	proxy.ModifyResponse = modifyResponse
	proxy.ServeHTTP(w, r)
}

func modifyResponse(resp *http.Response) error {
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		//判断内容的content-encoding并解压
		var reader io.Reader
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			gzipReader, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer gzipReader.Close()
			reader = gzipReader
		case "br":
			brotliReader := brotli.NewReader(resp.Body)
			defer resp.Body.Close() // 确保响应体在完成后关闭
			reader = brotliReader
		case "deflate":
			reader = flate.NewReader(resp.Body)
			defer resp.Body.Close()
		default:
			reader = resp.Body // 没有或不支持的编码类型，直接读取
		}

		// 读取解压后的body
		decodedBody, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}
		injectedScript := fmt.Sprintf(`
		<script>
		var pathParts = window.location.pathname.split('/');
		var containerID;
		if (pathParts.length > 1) {
			if (pathParts[1].length == "0f27e486215643f62403a9f7d97a620b12f24667a93131db3838aff6f520c0de".length){
				containerID = pathParts[1]; 
				var ws = new WebSocket("ws://" + window.location.host + "/appws");
				ws.onopen = function() {
					console.log("WebSocket connected");
					setInterval(function() {
						ws.send(JSON.stringify({ action: "updateTTL", containerID: containerID }));
					}, 60000);
				};
				ws.onmessage = function(evt) {
					console.log("Server response: " + evt.data);
				};
				ws.onerror = function(err) {
				   console.error('WebSocket encountered error: ', err.message, 'Closing socket');
				   ws.close();
				};
			}
			else
				console.error("Container ID not found in URL");
		} else {
			console.error("Container ID not found in URL");
		}
		</script>
		`)
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			decodedBody = injectScriptIntoHtml(decodedBody, injectedScript)
		}
		resp.Body = io.NopCloser(bytes.NewReader(decodedBody))
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
	}
	return nil
}
func injectScriptIntoHtml(bodyBytes []byte, INJECT_FILE string) []byte {
	re := regexp.MustCompile(`(<html.*?[^>]*>)`)
	bodyBytes = re.ReplaceAllFunc(bodyBytes, func(i []byte) []byte {
		return []byte(string(i) + INJECT_FILE)
	})
	return bodyBytes
}

func getContainerIP(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	dbRes := &ContainerInfo{}
	err := Db.Find(dbRes, &ContainerInfo{ContainerId: containerID}).Error
	if err != nil {
		return "", err
	}
	return dbRes.IP, nil
}

type ContainerInfo struct {
	ID          int64 `gorm:"primaryKey"`
	ContainerId string
	IP          string
	Port        string
	MinPort     int `gorm:"min_port"`
	ExpireAt    time.Time
}

// 检查新生成的端口范围是否与数据库中的记录冲突
func checkPortRangeConflict(db *gorm.DB, newMinPort, rangeSize int) (bool, error) {
	var conflicts int64
	newMaxPort := newMinPort + rangeSize - 1

	err := db.Model(&ContainerInfo{}).Where("? BETWEEN min_port AND min_port + ? - 1 OR ? BETWEEN min_port AND min_port + ? - 1",
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
func ReadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the YAML content into Config struct
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

var Db *gorm.DB
var ttl int
var checkInterval int
var wsUpdateInterval int

func main() {
	MapInfo.M = make(map[string]string, 0)
	db, err := initDB()
	if err != nil {
		fmt.Println("err")
		panic(err.Error())
	}
	Db = db
	config, err := ReadConfig("config.yml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	ttl = config.TTLMinutes
	checkInterval = config.CheckIntervalSeconds
	wsUpdateInterval = config.wsUpdateIntervalSeconds
	//根据间隔时间定时检查ttl
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Println("定时检查ttl")
				checkAndDeleteExpiredContainers()
			}
		}
	}()
	router := mux.NewRouter()
	router.HandleFunc("/start", startContainer).Methods(http.MethodGet)
	router.HandleFunc("/stop", stopContainer).Methods(http.MethodPost)
	router.HandleFunc("/appws", serveWs)
	router.PathPrefix("/").HandlerFunc(dynamicProxy)

	fmt.Println("Starting server on port 18083")
	if err := http.ListenAndServe(":18083", router); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

// ws服务
func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer ws.Close()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read:", err)
			break
		}
		log.Printf("recv: %s", message)

		// 解析消息
		var msg struct {
			Action      string `json:"action"`
			ContainerID string `json:"containerID"`
		}
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		if msg.Action == "updateTTL" {
			log.Println("Updating TTL for container", msg.ContainerID)
			// 更新容器的 TTL
			updateContainerTTL(msg.ContainerID)
		}

		if err := ws.WriteMessage(websocket.TextMessage, []byte("TTL updated for "+msg.ContainerID)); err != nil {
			log.Println("Write:", err)
			break
		}
	}
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
	// 根据网络类型选择合适的函数
	if network == "udp" || network == "udp4" || network == "udp6" {
		// UDP端口监听
		addr := fmt.Sprintf(":%d", port)
		conn, err := net.ListenPacket(network, addr)
		if err != nil {
			fmt.Println("Listen UDP Port err:", err.Error())
			return false
		}
		defer conn.Close()
	} else {
		// TCP端口监听
		listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Println("Listen TCP Port err:", err.Error())
			return false
		}
		defer listener.Close()
	}

	return true
}
