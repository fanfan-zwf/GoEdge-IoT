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

		_, ok = drive_value_list[d.Drive]
		if !ok {
			drive_value_list[d.Drive] = []fullConfig.Value_type{}
		}

		drive_value_list[d.Drive] = append(drive_value_list[d.Drive], value_list...)
	}

	for i, value := range drive_value_list {
		Write_value_mu.Lock()
		v, ok := Write_value[i]
		Write_value_mu.Unlock()
		if ok {
			log.Printf("INFO 写值-> %+v\n", value)
			err := (*v)(value)
			if err != nil {
				return err
			}
		} else {
			err := fmt.Errorf("ERROR map找不到驱动%d ", i)
			log.Print(err)
			return err
		}
	}

	return nil
}

// 变化更新 订阅 接收
func Write_value_Subscriber(p map[string]tag_drive_map_value, value Write_value_func_type) error {
	Write_value_mu.Lock()
	defer Write_value_mu.Unlock()

	for i, v := range p {
		tag_drive_map[i] = v
		_, ok := Write_value[v.Drive]
		if ok {
			continue
		}
		Write_value[v.Drive] = &value
	}

	return nil
}

// 变化更新 订阅 接收 mysql配置
func Write_value_Subscriber_mysqlconfig(cfg fullConfig.FullConfig_type, RW_Cancel map[string]bool, value Write_value_func_type) error {
	p := make(map[string]tag_drive_map_value)
	for _, v := range cfg.Points {
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

func init() {

}
