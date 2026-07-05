/*
* 日期: 2026-05-31 1:57
* 作者: 范范zwf
* 描述: 驱动管理器 统一管理所有驱动的查询、加载、启动，核心优化：先判断驱动是否支持，再查询点位配置，避免不必要的数据库查询和错误日志
 */

package manager

import (
	"main/IO/flexem_flexem"
	"main/IO/flexem_mqtt"
	"main/IO/manager/fullConfig"
	"main/db/mysql"

	"log"
	"sync"

	"errors"
	"fmt"
)

// 最关键：驱动管理器（支持 N 个驱动）

type DriverManager struct {
	drivers         map[uint]fullConfig.Driver // key = 驱动ID
	mu              sync.RWMutex
	shuttingDown    map[uint]bool // 正在关闭的驱动ID集合
	shutdownMu      sync.Mutex    // 保护 shuttingDown map
}

var Manager *DriverManager

func InitManager() {
	Manager = &DriverManager{
		drivers:      make(map[uint]fullConfig.Driver),
		shuttingDown: make(map[uint]bool),
	}
}

// 传建驱动
func (m *DriverManager) CreateDriver(cfg fullConfig.FullConfig_type) (fullConfig.Driver, error) {
	var driver fullConfig.Driver

	switch cfg.Drive.Type {
	case mysql.Mqtt__Type_Flexem_Mqtt:
		driver = &flexem_mqtt.Flexem_Mqtt{}
	case mysql.Mqtt__Type_Flexem_FlexEm:
		driver = &flexem_flexem.Flexem_FlexEm{}
	default:
		return nil, errors.New("不支持的驱动类型: " + cfg.Drive.Type)
	}

	// 自动加载配置
	err := driver.LoadConfig(cfg)
	if err != nil {
		return nil, err
	}

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

// ShutdownDriver 优雅关闭指定驱动
// 参数 id: 驱动ID（mqtt id）
// 返回: error - 关闭过程中的错误
func (m *DriverManager) ShutdownDriver(id uint) error {
	if id == 0 {
		return fmt.Errorf("无效的驱动ID")
	}

	// 1. 标记为正在关闭状态
	m.shutdownMu.Lock()
	if m.shuttingDown[id] {
		m.shutdownMu.Unlock()
		return fmt.Errorf("驱动 %d 已在关闭中", id)
	}
	m.shuttingDown[id] = true
	m.shutdownMu.Unlock()

	log.Printf("INFO 开始关闭驱动 ID:%d", id)

	// 2. 获取驱动实例
	m.mu.RLock()
	driver, exists := m.drivers[id]
	m.mu.RUnlock()

	if !exists {
		// 驱动不存在，直接清理关闭状态
		m.shutdownMu.Lock()
		delete(m.shuttingDown, id)
		m.shutdownMu.Unlock()
		return fmt.Errorf("驱动 %d 不存在", id)
	}

	// 3. 根据驱动类型执行关闭逻辑
	var err error
	switch d := driver.(type) {
	case *flexem_mqtt.Flexem_Mqtt:
		err = d.Stop() // 假设 Flexem_Mqtt 有 Stop 方法
	case *flexem_flexem.Flexem_FlexEm:
		err = d.Stop() // 假设 Flexem_FlexEm 有 Stop 方法
	default:
		err = fmt.Errorf("不支持的驱动类型")
	}

	// 4. 如果关闭成功，从管理器中移除驱动
	if err == nil {
		m.mu.Lock()
		delete(m.drivers, id)
		m.mu.Unlock()

		// 5. 确认正常关闭，从 shuttingDown map 中删除
		m.shutdownMu.Lock()
		delete(m.shuttingDown, id)
		m.shutdownMu.Unlock()

		log.Printf("INFO 驱动 %d 已成功关闭并清理", id)
	} else {
		log.Printf("ERROR 驱动 %d 关闭失败: %s", id, err)
		// 关闭失败时保留 shuttingDown 状态，便于后续重试或排查
	}

	return err
}

// IsShuttingDown 检查驱动是否正在关闭中
func (m *DriverManager) IsShuttingDown(id uint) bool {
	m.shutdownMu.Lock()
	defer m.shutdownMu.Unlock()
	return m.shuttingDown[id]
}

// GetShuttingDownDrivers 获取所有正在关闭的驱动ID列表
func (m *DriverManager) GetShuttingDownDrivers() []uint {
	m.shutdownMu.Lock()
	defer m.shutdownMu.Unlock()

	ids := make([]uint, 0, len(m.shuttingDown))
	for id := range m.shuttingDown {
		ids = append(ids, id)
	}
	return ids
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
