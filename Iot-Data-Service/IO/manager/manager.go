/*
* 日期: 2026-05-31 1:57
* 作者: 范范zwf
* 描述: 驱动管理器 统一管理所有驱动的查询、加载、启动，核心优化：先判断驱动是否支持，再查询点位配置，避免不必要的数据库查询和错误日志
 */

package manager

import (
	"main/IO/manager/fullConfig"
	"sync"

	"errors"
	"fmt"
)

// 最关键：驱动管理器（支持 N 个驱动）

type DriverManager struct {
	drivers map[uint]fullConfig.Driver // key = 驱动Name
	mu      sync.RWMutex
}

var Manager *DriverManager

func InitManager() {
	Manager = &DriverManager{
		drivers: make(map[uint]fullConfig.Driver),
	}
}

// 传建驱动
func (m *DriverManager) CreateDriver(cfg fullConfig.FullConfig_type) (fullConfig.Driver, error) {
	var driver fullConfig.Driver

	switch cfg.Drive.Type {

	default:
		return nil, errors.New("不支持的驱动类型: " + cfg.Drive.Type)
	}

	// 自动加载配置
	// err := driver.LoadConfig(cfg)
	// if err != nil {
	// 	return nil, err
	// }

	// 存入管理器
	m.mu.Lock()
	m.drivers[cfg.Drive.Id] = driver
	m.mu.Unlock()

	return driver, nil
}

// 根据驱动名获取任意驱动
func (m *DriverManager) GetDriver(id uint) (fullConfig.Driver, error) {
	if id == 0 {
		return nil, fmt.Errorf("无效的驱动ID")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.drivers[id]
	if !ok {
		return nil, fmt.Errorf("不存在的驱动 点位id=%d", id)
	}
	return d, nil
}

// func main() {
// 	// 1. 初始化管理器
// 	InitManager()

// 	// ==================== 加载 第一个驱动：TCP ====================
// 	config1 := fullConfig.FullConfig_type{
// 		Drive: mysql.Drive_Config_type{
// 			Id:     1,
// 			Config: "192.168.1.1:502",
// 			Name:   "tcp_driver_1",
// 			Type:   "Modbus_Tcp",
// 		},
// 	}
// 	Manager.CreateDriver(config1)

// 	// ==================== 加载 第二个驱动：MQTT ====================
// 	config2 := fullConfig.FullConfig_type{
// 		Drive: mysql.Drive_Config_type{},
// 	}
// 	Manager.CreateDriver(config2)

// 	// ==================== 获取 TCP 驱动 ====================
// 	driver1, _ := Manager.GetDriver(1)
// 	fmt.Println(driver1.GetDriveInfo().Name)

// 	// 转成 TCP 驱动才能拿到私有配置（关键）
// 	_, ok := driver1.(*TCPDriver)
// 	if ok {

// 	}

// 	// ==================== 获取 MQTT 驱动 ====================
// 	driver2, _ := Manager.GetDriver(2)
// 	mqttDriver, ok := driver2.(*MQTTDriver)
// 	if ok {
// 		fmt.Println("MQTT Broker:", mqttDriver.MQTTConfig.Broker)
// 	}
// }
