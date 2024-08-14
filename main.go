package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

const (
	FileURL    = "fileUrl"
	ByteLen    = 16
	UserId     = "uid"
	Connection = "Connection"
	Upgrade    = "Upgrade"
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

	env := []string{
		"NEKO_SCREEN=1920x1080@60",
		"NEKO_PASSWORD=rib",
		"NEKO_PASSWORD_ADMIN=rbi",
		"NEKO_EPR=56000-56100",
		"NEKO_NAT1TO1=202.63.172.204",
	}
	portBindings := map[nat.Port][]nat.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "8080"}},
	}

	for i := 57000; i <= 57100; i++ {
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
	}, nil, nil, "neko_user_"+req.UserId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Docker container: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		http.Error(w, "Failed to start Docker container", http.StatusInternalServerError)
		return
	}

	cmd := fmt.Sprintf("url=%s && filename=$(basename $url) && wget -O \"$filename\" $url && wps \"$filename\"", req.FileUrl)
	execConfig := container.ExecOptions{
		Cmd:          strslice.StrSlice{"bash", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}

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
	pathToProxy := "/" + strings.Join(parts[2:], "/")
	port, err := getContainerPort(containerID)
	if err != nil {
		http.Error(w, "Failed to get container port", http.StatusInternalServerError)
		return
	}
	target := "http://127.0.0.1:" + port + pathToProxy
	fmt.Println(target)
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Failed to parse target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.Header = r.Header
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = pathToProxy
		req.Host = targetURL.Host
	}
	if IsWebsocketRequest(r) {
		handleWebSocket(w, r, targetURL)
	} else {
		proxy.ServeHTTP(w, r)
	}
}
func handleWebSocket(w http.ResponseWriter, r *http.Request, targetURL *url.URL) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Connect to the target WebSocket server
	targetConn, _, err := websocket.DefaultDialer.Dial(targetURL.String(), nil)
	if err != nil {
		http.Error(w, "Failed to connect to target WebSocket server", http.StatusInternalServerError)
		return
	}
	defer targetConn.Close()

	// Handle WebSocket traffic between client and server
	go func() {
		for {
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = targetConn.WriteMessage(messageType, msg)
			if err != nil {
				break
			}
		}
	}()

	for {
		messageType, msg, err := targetConn.ReadMessage()
		if err != nil {
			break
		}
		err = conn.WriteMessage(messageType, msg)
		if err != nil {
			break
		}
	}
}

func getContainerPort(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	return "8080", nil
}
func getContainerIP(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	return "8080", nil
}

func initDB() (*sql.DB, error) {
	dbPath := "./containers.db"
	createTable := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		createTable = true
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if createTable {
		createTableSQL := `CREATE TABLE IF NOT EXISTS port_mappings (
            container_id TEXT PRIMARY KEY,
            host_port TEXT NOT NULL
        );`
		if _, err = db.Exec(createTableSQL); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func main() {
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
