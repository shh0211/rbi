package models

import "time"

type Automation struct {
	AutomationID int       `gorm:"primaryKey;autoIncrement"`
	UserID       int       `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name         string    `gorm:"type:text"`
	Description  string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	Actions      []Action  `gorm:"foreignKey:AutomationID"` // 关联的 Actions
}

type Action struct {
	ActionID     int    `gorm:"primaryKey;autoIncrement"`
	AutomationID int    `gorm:"index;foreignKey;references:AutomationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Sequence     int    `gorm:"not null"` // 动作执行顺序，非空
	ActionType   string `gorm:"type:text;not null"`
	Selector     string `gorm:"size:255"` // CSS选择器
	Value        string `gorm:"size:255"` // 发送的键值
	URL          string `gorm:"size:255"` // 导航URL
}

func init() {
	RegisterModel(&Automation{})
	RegisterModel(&Action{})
}
