/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"main/IO/manager/fullConfig"

	"fmt"
	"log"
	"sync"
)

/*
******************写入******************
 */
type tag_drive_map_value struct {
	Drive uint
	Type  string
}

type Write_value_func_type func([]fullConfig.Value_type) (err error)

var (
	tag_drive_map map[string]tag_drive_map_value

	Write_value    map[uint]*Write_value_func_type
	Write_value_mu sync.Mutex
)

func init() {
	tag_drive_map = make(map[string]tag_drive_map_value)
	Write_value = make(map[uint]*Write_value_func_type)
}

// 变化更新 发布 发送
func Write_value_Publisher(value_list []fullConfig.Value_type) error {
	if len(value_list) == 0 {
		return nil
	}

	// 优化：先收集所有驱动的数据，再批量处理
	drive_value_list := make(map[uint][]fullConfig.Value_type)
	
	for _, v := range value_list {
		d, ok := tag_drive_map[v.Tag]
		if !ok {
			log.Printf("ERROR 不存在的标识符 %s", v.Tag)
			return fmt.Errorf("不存在的标识符或者不是一个可写的点位 %s", v.Tag)
		}

		if d.Type != v.Type {
			log.Printf("ERROR 类型不匹配 %s ,配置类型 %s ,传递类型： %s", v.Tag, d.Type, v.Type)
			return fmt.Errorf("类型不匹配 %s ,配置类型 %s ,传递类型： %s", v.Tag, d.Type, v.Type)
		}

		// 修复Bug: 应该追加单个值 v，而不是整个列表 value_list
		drive_value_list[d.Drive] = append(drive_value_list[d.Drive], v)
	}

	// 优化：一次性获取所有需要的驱动函数引用，避免循环内频繁加锁
	Write_value_mu.Lock()
	drive_funcs := make(map[uint]*Write_value_func_type)
	for driveID := range drive_value_list {
		if v, ok := Write_value[driveID]; ok {
			drive_funcs[driveID] = v
		}
	}
	Write_value_mu.Unlock()

	// 批量执行写入操作
	for driveID, values := range drive_value_list {
		v, ok := drive_funcs[driveID]
		if !ok {
			err := fmt.Errorf("ERROR map找不到驱动%d ", driveID)
			log.Print(err)
			return err
		}
		
		log.Printf("INFO 写值-> %+v\n", values)
		err := (*v)(values)
		if err != nil {
			return err
		}
	}

	return nil
}

// 变化更新 订阅 接收
func Write_value_Subscriber(p map[string]tag_drive_map_value, value Write_value_func_type) error {
	// 修复Bug: 遵循"配置更新映射与注册分离规范"
	// 第一步：收集所有涉及的驱动ID
	driveIDs := make(map[uint]bool)
	
	Write_value_mu.Lock()
	defer Write_value_mu.Unlock()
	
	// 第二步：点位映射应始终更新，不受驱动是否存在影响
	for tag, v := range p {
		tag_drive_map[tag] = v
		driveIDs[v.Drive] = true
	}
	
	// 第三步：驱动函数注册仅在不存在时才进行
	for driveID := range driveIDs {
		if _, ok := Write_value[driveID]; !ok {
			Write_value[driveID] = &value
		}
	}

	return nil
}

// 变化更新 订阅 接收 mysql配置
func Write_value_Subscriber_mysqlconfig(cfg fullConfig.FullConfig_type, RW_Cancel map[string]bool, value Write_value_func_type) error {
	// 优化：先收集所有需要更新的映射，然后批量处理
	p := make(map[string]tag_drive_map_value)
	
	for _, v := range cfg.Points {
		if !RW_Cancel[v.RW_Cancel] {
			continue
		}

		p[v.Tag] = tag_drive_map_value{
			Drive: v.Drive.Id,
			Type:  v.Value_Type,
		}
	}
	
	return Write_value_Subscriber(p, value)
}

func init() {

}
