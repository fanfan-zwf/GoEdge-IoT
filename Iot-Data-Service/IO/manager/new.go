package manager

import (
	"fmt"
	"log"
	"main/IO/flexem_flexem"
	"main/IO/flexem_mqtt"
	"main/IO/manager/fullConfig"
	"main/db/db_point"
	"main/db/mysql"
	"time"
)

func Initialize(driveConfig mysql.Mqtt__type) (err error) {
	var fullConfig fullConfig.FullConfig_type
	fullConfig.Drive = driveConfig
	// 2. 支持 → 才查询点位（你要的核心优化）
	fullConfig.Points, err = mysql.Mqtt_Points__Query([]uint{driveConfig.Id}, []string{}, []string{}, 0, 0)
	if err != nil {
		log.Printf("ERROR 点位查询失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
		return
	}

	var driver any
	driver, err = Manager.CreateDriver(fullConfig)
	if err != nil {
		log.Printf("ERROR 创建驱动失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
		return
	}

	err = db_point.Alarm_Config_Subscriber_mysqlconfig(fullConfig)
	if err != nil {
		log.Printf("ERROR 创建报警失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
		return
	}

	switch driveConfig.Type {
	case mysql.Mqtt__Type_Flexem_Mqtt:
		flexem_mqtt_struct, ok := driver.(*flexem_mqtt.Flexem_Mqtt)
		if !ok {
			return
		}
		err = flexem_mqtt_struct.Start()
		if err != nil {
			log.Printf("ERROR 初始化驱动失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
			return
		}
		flexem_mqtt_struct.Callback_Push_External_Mappings = db_point.Collection_Publisher
		err = db_point.Write_value_Subscriber_mysqlconfig(fullConfig, map[string]bool{"R/W": true, "W": true}, flexem_mqtt_struct.Down)
		if err != nil {
			log.Printf("ERROR 订阅写点位失败 driveId:%d: %s", driveConfig.Id, err)
			return
		}
	case mysql.Mqtt__Type_Flexem_FlexEm:
		flexem_flexem_struct, ok := driver.(*flexem_flexem.Flexem_FlexEm)
		if !ok {
			return
		}
		err = flexem_flexem_struct.Start()
		if err != nil {
			log.Printf("ERROR 初始化驱动失败 :%d: %s", driveConfig.Id, err)
			return
		}
		flexem_flexem_struct.Callback_Push_External_Mappings = db_point.Collection_Publisher
	default:
		log.Printf("WARN 未知驱动类型: %s, 驱动ID: %d", driveConfig.Type, driveConfig.Id)
	}
	return
}

// InitializeDrivers 初始化采集器下所有支持的驱动（查询+加载+启动）
func InitializeDrivers() (err error) {
	// 1. 查询采集器
	var mqttConfigs []mysql.Mqtt__type

	mqttConfigs, err = mysql.Mqtt__Query([]uint{}, []string{}, 0, 0)
	if err != nil {
		return
	}

	InitManager() // 初始化驱动管理器

	// 3. 遍历驱动
	for _, driveConfig := range mqttConfigs {
		Initialize(driveConfig)
	}

	return nil
}
func New() (err error) {
	time.Sleep(1 * time.Second)
	InitializeDrivers()

	return
}

// ShutdownDriver 关闭指定驱动（优雅关闭）
// 参数 id: 驱动ID（mqtt id）
// 返回: error - 关闭过程中的错误
func ShutdownDriver(id uint) error {
	if Manager == nil {
		return fmt.Errorf("驱动管理器未初始化")
	}
	return Manager.ShutdownDriver(id)
}

// IsShuttingDown 检查驱动是否正在关闭中
func IsShuttingDown(id uint) bool {
	if Manager == nil {
		return false
	}
	return Manager.IsShuttingDown(id)
}

// GetShuttingDownDrivers 获取所有正在关闭的驱动ID列表
func GetShuttingDownDrivers() []uint {
	if Manager == nil {
		return []uint{}
	}
	return Manager.GetShuttingDownDrivers()
}

// RestartDriver 重启指定驱动（先关闭，再重新查询配置并启动）
// 参数 id: 驱动ID（mqtt id）
// 返回: error - 重启过程中的错误
func RestartDriver(id uint) error {
	if Manager == nil {
		return fmt.Errorf("驱动管理器未初始化")
	}

	log.Printf("INFO 开始重启驱动 ID:%d", id)

	// 1. 先关闭驱动
	err := ShutdownDriver(id)
	if err != nil {
		log.Printf("WARN 关闭驱动 %d 时出错（可能驱动不存在）: %s", id, err)
		// 继续执行，尝试重新启动
	}

	// 2. 等待一小段时间，确保资源完全释放
	time.Sleep(100 * time.Millisecond)

	// 3. 从数据库重新查询驱动配置
	var driveConfig mysql.Mqtt__type
	driveConfigs, err := mysql.Mqtt__Query([]uint{id}, []string{}, 0, 0)
	if err != nil {
		return fmt.Errorf("查询驱动 %d 配置失败: %w", id, err)
	}

	if len(driveConfigs) == 0 {
		return fmt.Errorf("驱动 %d 配置不存在", id)
	}

	driveConfig = driveConfigs[0]

	// 4. 重新初始化驱动
	err = Initialize(driveConfig)
	if err != nil {
		return fmt.Errorf("重新初始化驱动 %d 失败: %w", id, err)
	}

	log.Printf("INFO 驱动 %d 重启成功", id)
	return nil

}
