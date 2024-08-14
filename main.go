package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type StartRequest struct {
	UserId  int    `json:"userId"`
	FileUrl string `json:"fileUrl"`
}

type StopRequest struct {
	ContainerID string `json:"containerId"`
}

func startContainer(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

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
		"NEKO_EPR=52000-52100",
		"NEKO_NAT1TO1=47.57.190.193",
	}
	portBindings := map[nat.Port][]nat.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "9765"}},
	}
	// 映射57000-57100端口
	for i := 57000; i <= 57100; i++ {
		port := fmt.Sprintf("%d/udp", i)
		portBindings[nat.Port(port)] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", i)},
		}
	}
	// 创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "wps-latest",
		Env:   env,
	}, &container.HostConfig{
		ShmSize:      2 * 1024 * 1024 * 1024,
		PortBindings: portBindings,
		CapAdd:       strslice.StrSlice{"SYS_ADMIN"},
	}, nil, nil, "neko_user"+fmt.Sprintf("%d", req.UserId))
	if err != nil {
		http.Error(w, "Failed to create container", http.StatusInternalServerError)
		return
	}

	// 启动容器
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		http.Error(w, "Failed to start container", http.StatusInternalServerError)
		return
	}

	// 下载文件并打开
	fileUrl := req.FileUrl
	cmd := fmt.Sprintf("url=%s && filename=$(basename $url) && wget -O \"$filename\" $url && wps \"$filename\"", fileUrl)

	// 创建 exec 实例
	execConfig := container.ExecOptions{
		Cmd:          strslice.StrSlice{"bash", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := cli.ContainerExecCreate(ctx, resp.ID, execConfig)
	if err != nil {
		panic(err)
	}

	// 启动 exec 实例
	if err := cli.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println("Command executed inside container")

	// 返回容器ID
	response := map[string]string{
		"container_id": resp.ID,
	}
	json.NewEncoder(w).Encode(response)
}

func stopContainer(w http.ResponseWriter, r *http.Request) {
	var req StopRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Failed to create Docker client", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// 停止容器
	if err := cli.ContainerStop(ctx, req.ContainerID, container.StopOptions{}); err != nil {
		http.Error(w, "Failed to stop container", http.StatusInternalServerError)
		return
	}
	// 通过--rm启动的容器会自动删除
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Container stopped and removed successfully"))
}

func dynamicProxy(w http.ResponseWriter, r *http.Request) {
	// 假设路径格式为 /proxy/{container_id}/{path_to_proxy}
	parts := strings.SplitN(r.URL.Path, "/", 3)
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	containerID := parts[1]
	pathToProxy := "/" + parts[2]
	target := "http://127.0.0.1:" + getContainerPort(containerID) + pathToProxy

	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Failed to parse target URL", http.StatusInternalServerError)
		return
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}

func getContainerPort(containerID string) string {
	return "8080"
}

func main() {
	http.HandleFunc("/proxy/", dynamicProxy)
	http.HandleFunc("/start", startContainer)
	http.HandleFunc("/stop", stopContainer)
	http.ListenAndServe(":18080", nil)
}
