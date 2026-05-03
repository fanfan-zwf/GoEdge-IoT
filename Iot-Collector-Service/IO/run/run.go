package run

import (
	"main/IO/Modbus_Tcp"
	"main/Init"
	"main/app/mqtt_rpc"

	"fmt"
	"log"
	"time"
)

// InitializeDrivers 初始化采集器下所有支持的驱动（查询+加载+启动）
func InitializeDrivers() error {
	// 1. 查询采集器
	collectorInfos, err := mqtt_rpc.Collector_Info__Search_Field("Uuid", 1, Init.Config.APP.Uuid)
	if err != nil {
		return fmt.Errorf("ERROR 采集器查询失败: %w", err)
	}
	if len(collectorInfos) == 0 {
		return fmt.Errorf("ERROR 未找到采集设备 UUID: %s", Init.Config.APP.Uuid)
	}
	collectorInfo := collectorInfos[0]

	// 2. 查询驱动列表
	driveConfigs, err := mqtt_rpc.Drive_Config__Query(collectorInfo.Id, "", 0, 0)
	if err != nil {
		return fmt.Errorf("ERROR 驱动查询失败: %w", err)
	}

	// 优雅核心：驱动支持列表 + 初始化函数映射
	driverInitializers := map[string]func(cfg mqtt_rpc.IO_Points_Config_type) error{
		"Modbus_Tcp": Modbus_Tcp.New,
		// 以后加驱动只需要在这里加一行！
		// "OPC_UA":    Opcua.New,
		// "Serial":    Serial.New,
	}

	// 3. 遍历驱动
	for _, drive := range driveConfigs {
		// 1. 先判断是否支持（不支持直接跳过，不查点位）
		initFunc, ok := driverInitializers[drive.Type]
		if !ok {
			log.Printf("ERROR 跳过不支持的驱动 type:%s id:%d name:%s",
				drive.Type, drive.Id, drive.Name)
			continue
		}

		// 2. 支持 → 才查询点位（你要的核心优化）
		points, err := mqtt_rpc.Points_Config__Query(drive.Id, 0, 0)
		if err != nil {
			return fmt.Errorf("ERROR 点位查询失败 driveId:%d: %w", drive.Id, err)
		}

		// 构建 IO 配置
		ioConfig := mqtt_rpc.IO_Points_Config_type{
			Points: points,
			Drive:  drive,
		}

		// 3. 自动调用对应驱动初始化
		err = initFunc(ioConfig)
		if err != nil {
			log.Printf("ERROR 驱动启动失败 type:%s id:%d name:%s err:%v",
				drive.Type, drive.Id, drive.Name, err)
			return err
		}
	}

	return nil
}
func New() (err error) {
	time.Sleep(1 * time.Second)
	InitializeDrivers()
	return
}
