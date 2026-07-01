/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"fmt"
	"log"
	"main/IO/manager/fullConfig"
	"sync"
)

/*
******************写入******************
 */
type tag_drive_map_value struct {
	Drive uint   // 驱动ID
	Type  string // 数据类型
}

type Write_value_func_type func([]fullConfig.Value_type) error

var (
	tag_drive_map map[string]tag_drive_map_value
	tag_drive_mu  sync.RWMutex // ✅ 新增：保护 tag_drive_map 的锁

	Write_value    map[uint]*Write_value_func_type
	Write_value_mu sync.RWMutex // ✅ 改为 RWMutex
)

func init() {
	tag_drive_map = make(map[string]tag_drive_map_value)
	Write_value = make(map[uint]*Write_value_func_type)
}

// 变化更新 发布 发送（修复 Bug + 性能优化）
func Write_value_Publisher(value_list []fullConfig.Value_type) error {
	if len(value_list) == 0 {
		return nil
	}

	// 按驱动分组
	drive_value_list := make(map[uint][]fullConfig.Value_type)
	
	for _, v := range value_list {
		// 安全校验
		if v.Tag == "" {
			log.Printf("WARN 跳过无效点位: Tag为空")
			continue
		}

		tag_drive_mu.RLock() // ✅ 使用读锁查询
		d, ok := tag_drive_map[v.Tag]
		tag_drive_mu.RUnlock()
		
		if !ok {
			err := fmt.Errorf("不存在的标识符或者不是一个可写的点位: %s", v.Tag)
			log.Printf("ERROR %s", err)
			return err
		}

		if d.Type != v.Type {
			err := fmt.Errorf("类型不匹配 %s, 配置类型: %s, 传递类型: %s", v.Tag, d.Type, v.Type)
			log.Printf("ERROR %s", err)
			return err
		}

		// ✅ 修复：添加单个值 v，而不是整个列表 value_list
		drive_value_list[d.Drive] = append(drive_value_list[d.Drive], v)
	}

	if len(drive_value_list) == 0 {
		return nil
	}

	// 批量执行写入（一次性获取所有驱动函数）
	Write_value_mu.RLock()
	funcs := make(map[uint]*Write_value_func_type, len(drive_value_list))
	for driveID := range drive_value_list {
		if f, ok := Write_value[driveID]; ok {
			funcs[driveID] = f
		}
	}
	Write_value_mu.RUnlock()

	// 执行写入（锁外执行，减少锁持有时间）
	for driveID, values := range drive_value_list {
		f, ok := funcs[driveID]
		if !ok {
			err := fmt.Errorf("找不到驱动 %d 的写入函数", driveID)
			log.Printf("ERROR %s", err)
			return err
		}

		if f == nil { // ✅ 空指针检查
			err := fmt.Errorf("驱动 %d 的写入函数为nil", driveID)
			log.Printf("ERROR %s", err)
			return err
		}

		log.Printf("INFO 写值-> 驱动ID=%d, 点位数量=%d", driveID, len(values))
		if err := (*f)(values); err != nil {
			log.Printf("ERROR 驱动 %d 写入失败: %v", driveID, err)
			return err
		}
	}

	return nil
}

// 变化更新 订阅 接收（修复逻辑错误）
func Write_value_Subscriber(p map[string]tag_drive_map_value, value Write_value_func_type) error {
	if value == nil {
		return fmt.Errorf("写入函数不能为nil")
	}

	if len(p) == 0 {
		return nil
	}

	// 先收集需要更新的驱动ID
	driveIDs := make(map[uint]bool)
	
	tag_drive_mu.Lock()
	defer tag_drive_mu.Unlock()
	
	for tag, v := range p {
		// 安全校验
		if tag == "" || v.Drive == 0 {
			log.Printf("WARN 跳过无效配置: Tag=%s, Drive=%d", tag, v.Drive)
			continue
		}
		
		tag_drive_map[tag] = v
		driveIDs[v.Drive] = true
	}

	// 批量注册驱动函数（只加一次锁）
	if len(driveIDs) > 0 {
		Write_value_mu.Lock()
		defer Write_value_mu.Unlock()
		
		for driveID := range driveIDs {
			if _, ok := Write_value[driveID]; !ok {
				Write_value[driveID] = &value
			}
		}
	}

	return nil
}

// 变化更新 订阅 接收 mysql配置
func Write_value_Subscriber_mysqlconfig(cfg fullConfig.FullConfig_type, RW_Cancel map[string]bool, value Write_value_func_type) error {
	if len(cfg.Points) == 0 {
		return nil
	}

	p := make(map[string]tag_drive_map_value)
	for _, v := range cfg.Points {
		// 安全校验
		if v.Tag == "" {
			continue
		}

		if !RW_Cancel[v.RW_Cancel] {
			continue
		}

		p[v.Tag] = tag_drive_map_value{
			Drive: v.Mqtt_Id,
			Type:  v.Value_Type,
		}
	}
	
	return Write_value_Subscriber(p, value)
}
// 删除点位映射（新增功能）
func RemoveTagDriveMapping(tag string) {
	if tag == "" {
		return
	}
	
	tag_drive_mu.Lock()
	defer tag_drive_mu.Unlock()
	delete(tag_drive_map, tag)
}

// 获取所有点位映射快照（新增功能）
func GetAllTagDriveMappings() map[string]tag_drive_map_value {
	tag_drive_mu.RLock()
	defer tag_drive_mu.RUnlock()
	
	result := make(map[string]tag_drive_map_value, len(tag_drive_map))
	for k, v := range tag_drive_map {
		result[k] = v
	}
	return result
}

func init() {
	// 初始化逻辑（如果需要）
}
