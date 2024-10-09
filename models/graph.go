package models

import (
	"encoding/json"
)

// AutomationData 用于存储自动化脚本中的图结构数据
type GraphData struct {
	ID           int             `gorm:"primaryKey;autoIncrement"`
	AutomationID int             `json:"automation_id" gorm:"index;foreignKey:AutomationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Data         json.RawMessage `json:"data"` // 使用 RawMessage 来存储任意 JSON 数据
}

func init() {
	RegisterModel(&GraphData{})
}
