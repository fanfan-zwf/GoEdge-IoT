package manager

import (
	"main/db/db_point"
	"main/db/mysql"

	"fmt"
	"log"
	"time"
)

// InitializeDrivers 初始化采集器下所有支持的驱动（查询+加载+启动）
// 配置来源：从 MySQL 读取（启动前已通过 syncConfigOnStartup 同步最新配置）
func InitializeDrivers() error {
	// 从 MySQL 获取驱动配置
	driveConfigs, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		return fmt.Errorf("驱动配置获取失败: %w", err)
	}
	if len(driveConfigs) == 0 {
		log.Printf("WARN 当前采集器下无驱动配置")
		return nil
	}

	// 2. 从 MySQL 获取点位配置
	pointConfigs, err := mysql.Point_Config__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("点位配置获取失败: %w", err)
	}

	// 3. 按驱动ID分组点位配置（避免每个驱动都遍历全量点位）
	pointsByDriveId := make(map[uint][]mysql.Point_Config_Query_type)
	for _, p := range pointConfigs {
		pointsByDriveId[p.Drive_Id] = append(pointsByDriveId[p.Drive_Id], p)
	}

	InitManager() // 初始化驱动管理器

	// 4. 遍历驱动，逐个初始化
	var firstErr error
	for _, driveConfig := range driveConfigs {
		if err := initSingleDriver(driveConfig, pointsByDriveId[driveConfig.Id]); err != nil {
			log.Printf("ERROR 驱动 id=%d 初始化失败: %v", driveConfig.Id, err)
			if firstErr == nil {
				firstErr = err
			}
			// 单个驱动失败不影响其他驱动初始化
			continue
		}
	}

	return firstErr
}

// initSingleDriver 初始化单个驱动：创建 → New → Connect
func initSingleDriver(drive mysql.Drive_Config_Query_type, points []mysql.Point_Config_Query_type) error {
	// 1. 创建驱动实例
	_, err := CreateDriver(drive.Type, drive.Id)
	if err != nil {
		return fmt.Errorf("创建驱动失败: %w", err)
	}

	// 2. 初始化驱动配置
	if err := DriveNew(drive.Id, drive, points); err != nil {
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	log.Printf("INFO 驱动 id=%d type=%s 初始化完成，点位数=%d", drive.Id, drive.Type, len(points))

	// 3. 连接驱动并绑定数据回调（会启动轮询 goroutine）
	if err := DriveConnect(drive.Id, db_point.Collection_Publisher); err != nil {
		return fmt.Errorf("连接驱动失败: %w", err)
	}

	return nil
}

func Start() error {
	time.Sleep(1 * time.Second)
	return InitializeDrivers()
}
