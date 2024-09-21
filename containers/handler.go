package containers

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
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net"
	"net/http"
	config2 "rbi/config"
	"rbi/models"
	"rbi/sqlite"
	"time"
)

func RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/start", startContainer).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/stop", stopContainer).Methods(http.MethodPost)
	router.HandleFunc("/list", listContainer).Methods(http.MethodGet)
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

var ttl int
var checkInterval int
var Db = sqlite.Db

func InitTTLCheck() {
	ttl = config2.Config.TTLMinutes
	checkInterval = config2.Config.CheckIntervalSeconds
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
}

type StartRequest struct {
	UserId  string `json:"userId"`
	FileUrl string `json:"fileUrl"`
}

type StopRequest struct {
	ContainerID string `json:"containerId"`
}

func listContainer(w http.ResponseWriter, r *http.Request) {
	var containers []models.ContainerInfo
	if err := Db.Find(&containers).Error; err != nil {
		log.Fatal("failed to retrieve data: ", err)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(containers); err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		log.Println("failed to encode data: ", err)
		return
	}
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
	containerInfo := &models.ContainerInfo{
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
	if resp.ID == "" {
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
	if err := tx.Where("container_id = ?", req.ContainerID).Delete(&models.ContainerInfo{}).Error; err != nil {
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

// 检查并删除过期容器
func checkAndDeleteExpiredContainers() {
	var expiredContainers []models.ContainerInfo
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

// 检查新生成的端口范围是否与数据库中的记录冲突
func checkPortRangeConflict(db *gorm.DB, newMinPort, rangeSize int) (bool, error) {
	var conflicts int64
	newMaxPort := newMinPort + rangeSize - 1

	err := db.Model(&models.ContainerInfo{}).Where("? BETWEEN min_port AND min_port + ? - 1 OR ? BETWEEN min_port AND min_port + ? - 1",
		newMinPort, rangeSize, newMaxPort, rangeSize).Count(&conflicts).Error

	if err != nil {
		return false, err
	}

	return conflicts == 0, nil
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
