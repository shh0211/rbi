package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/gorilla/mux"
	"net/http"
	"rbi/models"
	"rbi/sqlite"
	"strconv"
	"time"
)

func RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/automation/newScript", newScript).Methods(http.MethodPost)
	router.HandleFunc("/automation/getScripts", getScripts).Methods(http.MethodGet)
	router.HandleFunc("/automation/delScript", delScript).Methods(http.MethodPost)
	router.HandleFunc("/automation/updateAction", updateAction).Methods(http.MethodPost)
	router.HandleFunc("automation/runScript", runScript).Methods(http.MethodPost)
}

var Db = sqlite.Db

// 新建自动化脚本
func newScript(w http.ResponseWriter, r *http.Request) {
	var automation models.Automation
	if err := json.NewDecoder(r.Body).Decode(&automation); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := Db.Create(&automation).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("自动化脚本 %d 创建成功", automation.AutomationID)))
}

// 获取全部脚本
func getScripts(w http.ResponseWriter, r *http.Request) {
	var automations []models.Automation
	if err := Db.Preload("Actions").Find(&automations).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(automations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 删除某个自定义脚本
func delScript(w http.ResponseWriter, r *http.Request) {
	scriptIDStr := r.URL.Query().Get("id")
	scriptID, err := strconv.Atoi(scriptIDStr)
	if err != nil || scriptID <= 0 {
		http.Error(w, "无效的脚本ID", http.StatusBadRequest)
		return
	}

	if err := Db.Delete(&models.Automation{}, scriptID).Error; err != nil {
		http.Error(w, "删除脚本失败", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("自动化脚本 %d 删除成功", scriptID)))
}

type UpdateActionsRequest struct {
	AutomationID int             `json:"automation_id"`
	Actions      []models.Action `json:"actions"`
}

// 修改某条 Automation 记录中的 Actions
func updateAction(w http.ResponseWriter, r *http.Request) {
	// 解析请求体中的 AutomationID 和新的 Actions 数组
	var req UpdateActionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 查找指定的 Automation 记录
	var automation models.Automation
	if err := Db.Preload("Actions").First(&automation, req.AutomationID).Error; err != nil {
		http.Error(w, "无法找到指定的 Automation", http.StatusNotFound)
		return
	}

	// 开始事务
	tx := Db.Begin()

	// 删除当前 Automation 中的所有旧的 Actions
	if err := tx.Where("automation_id = ?", req.AutomationID).Delete(&models.Action{}).Error; err != nil {
		tx.Rollback()
		http.Error(w, "删除旧 Actions 失败", http.StatusInternalServerError)
		return
	}

	for i := range req.Actions {
		req.Actions[i].AutomationID = req.AutomationID
	}

	if err := tx.Create(&req.Actions).Error; err != nil {
		tx.Rollback()
		http.Error(w, "插入新 Actions 失败", http.StatusInternalServerError)
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		http.Error(w, "提交事务失败", http.StatusInternalServerError)
		return
	}

	// 成功返回
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Automation %d 的 Actions 更新成功", req.AutomationID)))
}

type RunScriptRequest struct {
	AutomationID int    `json:"automation_id"`
	RemoteURL    string `json:"remote_url"` // 浏览器调试地址
}

// 执行自动化脚本
func runScript(w http.ResponseWriter, r *http.Request) {
	var req RunScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var automation models.Automation
	if err := Db.Preload("Actions").First(&automation, req.AutomationID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	err := executeActions(automation.Actions, req.RemoteURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("执行脚本时发生错误: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Automation %d 的脚本执行成功", req.AutomationID)))
}

func executeActions(actions []models.Action, remoteURL string) error {
	// 连接到远程 Chrome 实例
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), remoteURL)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 创建任务列表
	var tasks chromedp.Tasks
	for _, action := range actions {
		switch action.ActionType {
		case "navigate":
			tasks = append(tasks, chromedp.Navigate(action.URL))
		case "waitVisible":
			tasks = append(tasks, chromedp.WaitVisible(action.Selector, chromedp.ByQuery))
		case "sendKeys":
			tasks = append(tasks, chromedp.SendKeys(action.Selector, action.Value, chromedp.ByQuery))
		case "click":
			tasks = append(tasks, chromedp.Click(action.Selector, chromedp.ByQuery))
		default:
			return fmt.Errorf("未知的 ActionType: %s", action.ActionType)
		}
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("执行 chromedp 任务失败: %v", err)
	}

	return nil
}
