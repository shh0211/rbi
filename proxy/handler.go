package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"rbi/config"
	"rbi/models"
	"rbi/sqlite"
	"regexp"
	"strings"
	"sync"
	"time"
)

func RegisterRoutes(router *mux.Router) {
	MapInfo.M = make(map[string]string, 0)
	ttl = config.Config.TTLMinutes
	router.HandleFunc("/appws", serveWs)
	router.PathPrefix("/").HandlerFunc(dynamicProxy)
}

type Mapping struct {
	M map[string]string
	sync.RWMutex
}

var MapInfo Mapping
var Db = sqlite.Db
var ttl int

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

func getContainerIP(containerID string) (string, error) {
	// Replace with actual logic to retrieve the container port from the database
	dbRes := &models.ContainerInfo{}
	err := Db.Find(dbRes, &models.ContainerInfo{ContainerId: containerID}).Error
	if err != nil {
		return "", err
	}
	return dbRes.IP, nil
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

// Define an upgrader to upgrade HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust this function if needed for origin checks
	},
}

// 更新容器ttl
func updateContainerTTL(containerID string) {
	result := Db.Model(&models.ContainerInfo{}).Where("container_id = ?", containerID).Update("expire_at", time.Now().Add(time.Duration(ttl)*time.Minute))
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
