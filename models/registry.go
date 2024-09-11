package models

var registry []interface{}

// RegisterModel 注册模型到全局列表
func RegisterModel(model interface{}) {
	registry = append(registry, model)
}

// GetAllModels 返回所有注册的模型
func GetAllModels() []interface{} {
	return registry
}
