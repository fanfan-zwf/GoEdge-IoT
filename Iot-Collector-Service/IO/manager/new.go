package manager

import (
	"main/IO/Modbus_Tcp"
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/app/mqtt"
	"main/db/db_point"
	"main/db/mysql"

	"fmt"
	"log"

	"time"
)

// InitializeDrivers 初始化采集器下所有支持的驱动（查询+加载+启动）
func InitializeDrivers() (err error) {
	// 1. 查询采集器
	var collectorInfos []mysql.Collector_Info_type
	collectorInfos, err = mqtt.Collector_Info__Search_Field("Uuid", 1, Init.Config.APP.Uuid)
	if err != nil {
		return fmt.Errorf("ERROR 采集器查询失败: %w", err)
	}
	if len(collectorInfos) == 0 {
		return fmt.Errorf("ERROR 未找到采集设备 UUID: %s", Init.Config.APP.Uuid)
	}
	collectorInfo := collectorInfos[0]

	// 2. 查询驱动列表
	driveConfigs, err := mqtt.Drive_Config__Query([]uint{collectorInfo.Id}, []string{}, 0, 0)
	if err != nil {
		return fmt.Errorf("ERROR 驱动查询失败: %w", err)
	}

	InitManager() // 初始化驱动管理器

	// 3. 遍历驱动
	for _, driveConfig := range driveConfigs {

		var fullConfig fullConfig.FullConfig_type
		fullConfig.Drive = driveConfig
		// 2. 支持 → 才查询点位（你要的核心优化）
		fullConfig.Points, err = mqtt.Points_Config__Query([]uint{}, []uint{driveConfig.Id}, 0, 0)
		if err != nil {
			log.Printf("ERROR 点位查询失败 driveId:%d: %s", driveConfig.Id, err)
			return
		}

		var driver any
		driver, err = Manager.CreateDriver(fullConfig)
		if err != nil {
			log.Printf("ERROR 创建驱动失败 driveId:%d: %s", driveConfig.Id, err)
			return
		}

		err = db_point.Alarm_Config_Subscriber_mysqlconfig(fullConfig)
		if err != nil {
			log.Printf("ERROR 创建报警失败 driveId:%d: %s", driveConfig.Id, err)
			return
		}

		switch driveConfig.Type {
		case "Modbus_Tcp":
			modbus_tcp_struct, ok := driver.(*Modbus_Tcp.Modbus_Tcp)
			if !ok {
				continue
			}
			err = modbus_tcp_struct.New()
			if err != nil {
				log.Printf("ERROR 初始化驱动失败 driveId:%d: %s", driveConfig.Id, err)
				return
			}
			err = modbus_tcp_struct.Connect()
			if err != nil {
				log.Printf("ERROR 连接驱动失败 driveId:%d: %s", driveConfig.Id, err)
				return
			}
			modbus_tcp_struct.Read_External_Mappings = db_point.Collection_Publisher
			err = db_point.Write_value_Subscriber_mysqlconfig(fullConfig, map[string]bool{"R/W": true, "W": true}, modbus_tcp_struct.Write)
			if err != nil {
				log.Printf("ERROR 订阅写点位失败 driveId:%d: %s", driveConfig.Id, err)
				return
			}
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
