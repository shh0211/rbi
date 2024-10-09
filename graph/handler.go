package graph

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"rbi/models"
	"rbi/sqlite"
	"strconv"
)

var Db = sqlite.Db

// RegisterRoutes 注册路由
func RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/graph/update", updateGraph).Methods(http.MethodPost)
	router.HandleFunc("/graph/get", getGraph).Methods(http.MethodGet)
}

// updateGraph 处理函数，如果没有记录就新建，如果有就删了重新加入一条
func updateGraph(w http.ResponseWriter, r *http.Request) {
	var graphData models.GraphData

	// 从请求体中解析传入的 JSON 数据
	if err := json.NewDecoder(r.Body).Decode(&graphData); err != nil {
		http.Error(w, "请求体解析失败", http.StatusBadRequest)
		return
	}

	// 查询数据库中是否已经存在该 AutomationID 的记录
	var existingGraph models.GraphData
	if err := Db.Where("automation_id = ?", graphData.AutomationID).First(&existingGraph).Error; err == nil {
		// 如果存在记录，删除该记录
		if err := Db.Delete(&existingGraph).Error; err != nil {
			http.Error(w, "删除旧图数据失败", http.StatusInternalServerError)
			return
		}
	}

	// 创建新的图数据记录
	if err := Db.Create(&graphData).Error; err != nil {
		http.Error(w, "新建图数据失败", http.StatusInternalServerError)
		return
	}

	// 返回成功信息
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("图结构更新成功, AutomationID: %d", graphData.AutomationID)))
}

// getGraph 处理函数，用于根据 AutomationID 获取图数据
func getGraph(w http.ResponseWriter, r *http.Request) {
	// 从查询参数中获取 AutomationID
	automationIDStr := r.URL.Query().Get("automation_id")
	automationID, err := strconv.Atoi(automationIDStr)
	if err != nil || automationID <= 0 {
		http.Error(w, "无效的 Automation ID", http.StatusBadRequest)
		return
	}

	// 查询与该 AutomationID 相关的图数据
	var graphData models.GraphData
	if err := Db.Where("automation_id = ?", automationID).First(&graphData).Error; err != nil {
		http.Error(w, "未找到对应的图数据", http.StatusNotFound)
		return
	}

	// 返回查询到的图数据
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(graphData); err != nil {
		http.Error(w, "返回图数据失败", http.StatusInternalServerError)
	}
}
