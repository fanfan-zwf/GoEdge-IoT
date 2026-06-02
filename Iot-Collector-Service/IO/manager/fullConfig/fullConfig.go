package fullConfig

import (
	"main/db/mysql"
	"sync"
	"time"
)

// FullConfig 驱动全配置（驱动配置 + 该驱动下的所有点位配置）
type FullConfig_type struct {
	Drive  mysql.Drive_Config_type
	Points []mysql.Points_Config_type
}

// 统一驱动接口（必须）
type Driver interface {
	LoadConfig(cfg FullConfig_type) error  // 加载配置（驱动 + 点位）
	GetDriveInfo() mysql.Drive_Config_type // 获取驱动配置信息
	GetPoints() []mysql.Points_Config_type // 获取点位配置信息
}

type BaseDriver struct {
	Config FullConfig_type
	mu     sync.Mutex
}

// 统一驱动接口（必须）
func (b *BaseDriver) LoadConfig(cfg FullConfig_type) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Config = cfg
	return nil
}

// 获取驱动配置信息
func (b *BaseDriver) GetDriveInfo() mysql.Drive_Config_type {
	return b.Config.Drive
}

// 获取点位配置信息
func (b *BaseDriver) GetPoints() []mysql.Points_Config_type {
	return b.Config.Points
}

type Value_type struct {
	Tag   string    // 点位名称
	Value any       // 点位值
	Type  string    // 输出类型
	Msg   string    // 状态信息
	Time  time.Time // 读取时间
}

func Read_Value(v Value_type) error {

	return nil
}
