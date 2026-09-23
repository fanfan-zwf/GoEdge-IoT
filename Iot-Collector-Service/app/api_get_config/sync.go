package api_get_config

import (
	"fmt"
	"log"
	"main/Init"
	"main/db/mysql"
	"time"
)

// appConfigKey_ConfigUpdateTime APP_Config 表中存储配置更新时间的 key
const appConfigKey_ConfigUpdateTime = "ConfigUpdateTime"

// getLastConfigUpdateTime 从 MySQL APP_Config 表读取上次配置同步时间
// 返回零值 time.Time 表示无记录（首次运行）
func getLastConfigUpdateTime() (time.Time, error) {
	str, err := mysql.APP_Config__GetByKet(appConfigKey_ConfigUpdateTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("读取上次同步时间失败: %w", err)
	}
	if str == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}, fmt.Errorf("解析上次同步时间失败: %w", err)
	}
	return t, nil
}

// setLastConfigUpdateTime 将本次配置同步时间写入 MySQL APP_Config 表
func setLastConfigUpdateTime(t time.Time) error {
	return mysql.APP_Config__SetByKet(appConfigKey_ConfigUpdateTime, t.Format(time.RFC3339))
}

// IsConfigUpdated 检查配置服务是否有更新
// 流程：调用 Collector_ConfigUpdate 获取服务端最新更新时间 → 与 MySQL 记录的上次同步时间比对
// 返回值：configUpdateTime 非零表示有更新，零值表示无更新或配置服务未启用
// 注意：请求遇到 401 时会自动刷新 token 重试（在 Collector_ConfigUpdate 内部处理）
func IsConfigUpdated() (configUpdateTime time.Time, err error) {
	if !Init.Config.Config_Service.Enable {
		return time.Time{}, nil
	}

	// 查询配置服务最新更新时间
	configUpdateTime, err = Collector_ConfigUpdate()
	if err != nil {
		return time.Time{}, fmt.Errorf("查询配置更新时间失败: %w", err)
	}

	// 从 MySQL 读取上次同步时间
	lastSyncTime, err := getLastConfigUpdateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("获取上次同步时间失败: %w", err)
	}

	// 服务端更新时间 <= 本地记录时间 → 配置未更新
	if !configUpdateTime.IsZero() && !lastSyncTime.IsZero() && !configUpdateTime.After(lastSyncTime) {
		log.Printf("INFO 配置未更新（上次同步时间: %s）", lastSyncTime.Format(time.RFC3339))
		return time.Time{}, nil
	}

	return configUpdateTime, nil
}

// SyncConfigFromService 从配置服务拉取全量配置并同步到 MySQL
// 参数 configUpdateTime：由 IsConfigUpdated 返回的服务端更新时间，同步成功后写入 MySQL
// 流程：获取驱动/点位配置 → 调用 __Sync 按 Id 比对增删改 → 记录同步时间
func SyncConfigFromService(configUpdateTime time.Time) error {
	// 从配置服务获取驱动配置
	driveConfigs, err := Collector_Drive_Config__Query()
	if err != nil {
		return fmt.Errorf("驱动配置获取失败: %w", err)
	}

	// 从配置服务获取点位配置
	pointConfigs, err := Collector_Point_Config__Query()
	if err != nil {
		return fmt.Errorf("点位配置获取失败: %w", err)
	}

	// 同步到 MySQL（__Sync 内部按 Id 比对，执行新增/更新/删除）
	if err := mysql.Drive_Config__Sync(driveConfigs); err != nil {
		return fmt.Errorf("驱动配置同步失败: %w", err)
	}
	if err := mysql.Point_Config__Sync(pointConfigs); err != nil {
		return fmt.Errorf("点位配置同步失败: %w", err)
	}

	// 同步成功后，记录本次时间到 MySQL
	err = setLastConfigUpdateTime(configUpdateTime)
	if err != nil {
		log.Printf("WARN 记录同步时间失败: %v", err)
	}

	log.Printf("INFO 配置服务同步完成：驱动 %d 条，点位 %d 条", len(driveConfigs), len(pointConfigs))
	return nil
}
