package models

import "time"

type User struct {
	UserID    uint   `gorm:"primaryKey;autoIncrement"` // 主键，自增
	Username  string `gorm:"unique"`                   // 用户名唯一
	Password  string
	IsAdmin   bool
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

func init() {
	RegisterModel(&User{})
}
