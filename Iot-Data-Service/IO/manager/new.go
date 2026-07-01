package manager

import (
	"log"
	"main/IO/flexem_flexem"
	"main/IO/flexem_mqtt"
	"main/IO/manager/fullConfig"
	"main/db/db_point"
	"main/db/mysql"
	"time"
)

// InitializeDrivers 初始化采集器下所有支持的驱动（查询+加载+启动）
func InitializeDrivers() (err error) {
	// 1. 查询采集器
	var mqttConfigs []mysql.Mqtt__type

	mqttConfigs, err = mysql.Mqtt__Query([]string{}, 0, 0)
	if err != nil {
		return
	}

	InitManager() // 初始化驱动管理器

	// 3. 遍历驱动
	for _, driveConfig := range mqttConfigs {

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
				continue
			}
			err = flexem_mqtt_struct.New()
			if err != nil {
				log.Printf("ERROR 初始化驱动失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
				return
			}
			err = flexem_mqtt_struct.Push()
			if err != nil {
				log.Printf("ERROR 连接驱动失败 Mqtt_Id:%d: %s", driveConfig.Id, err)
				return
			}
			flexem_mqtt_struct.Push_External_Mappings = db_point.Collection_Publisher
			err = db_point.Write_value_Subscriber_mysqlconfig(fullConfig, map[string]bool{"R/W": true, "W": true}, flexem_mqtt_struct.Down)
			if err != nil {
				log.Printf("ERROR 订阅写点位失败 driveId:%d: %s", driveConfig.Id, err)
				return
			}
		case mysql.Mqtt__Type_Flexem_FlexEm:
			flexem_flexem_struct, ok := driver.(*flexem_flexem.Flexem_FlexEm)
			if !ok {
				continue
			}
			err = flexem_flexem_struct.Start()
			if err != nil {
				log.Printf("ERROR 初始化驱动失败 :%d: %s", driveConfig.Id, err)
				return
			}
			flexem_flexem_struct.Push_External_Mappings = db_point.Collection_Publisher
		default:
			log.Printf("WARN 未知驱动类型: %s, 驱动ID: %d", driveConfig.Type, driveConfig.Id)
		}
	}

	return nil
}
func New() (err error) {
	time.Sleep(1 * time.Second)
	InitializeDrivers()

	return
}
