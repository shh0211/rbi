package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"rbi/models"
)

var Db *gorm.DB

func InitDB() {
	// 初始化 SQLite 数据库连接
	db, err := gorm.Open(sqlite.Open("rbi.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}
	// 获取所有注册的模型
	modelList := models.GetAllModels()

	// 迁移所有模型
	for _, model := range modelList {
		if err := db.AutoMigrate(model); err != nil {
			log.Fatalf("failed to migrate model %T: %v", model, err)
		}
	}

	Db = db
}
