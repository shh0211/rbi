package models

import "time"

type ContainerInfo struct {
	ID          int64 `gorm:"primaryKey"`
	ContainerId string
	IP          string
	Port        string
	UserID      int64 `gorm:"foreignKey:UserID"`
	MinPort     int   `gorm:"min_port"`
	ExpireAt    time.Time
}

func init() {
	RegisterModel(&ContainerInfo{})
}
