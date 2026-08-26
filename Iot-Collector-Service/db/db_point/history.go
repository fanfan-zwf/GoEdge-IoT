/*
* 日期: 2026.6.5 PM3:49
* 作者: 范范zwf
* 作用: 记录数据库
 */

package db_point

import (
	"fmt"
	"log"
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"sync"
	"time"
)

type History_Value_type struct {
	DeviceId string    // 设备id
	PointId  uint      // 点位id
	Time     time.Time // 记录时间

	Msg   string // 状态信息
	Value any    // 记录值
	Type  string // 值类型
}

type History_Config_type struct {
	DeviceId string // 设备id
	PointId  uint   // 点位id
	Config   string // 配置
}

var (
	History_Config      map[Config_key_type]History_Config_type
	History_Config_RWMu sync.RWMutex
)

func init() {
	History_Config = make(map[Config_key_type]History_Config_type)
}

// 读取配置
func History_Config__Query(key Config_key_type) (History_Config_type, bool) {
	History_Config_RWMu.RLock()
	defer History_Config_RWMu.RUnlock()
	v, ok := History_Config[key]
	return v, ok
}
func History_Config__Query_list(keys []Config_key_type) (r []History_Config_type) {
	for _, key := range keys {
		v, ok := History_Config__Query(key)
		if ok {
			r = append(r, v)
		}
	}
	return
}

// 增加配置
func History_Config__Add(configs []History_Config_type) error {

	// 优化：批量操作，只在循环外加一次锁，避免N次锁开销
	History_Config_RWMu.Lock()
	defer History_Config_RWMu.Unlock()

	for _, config := range configs {
		key := Config_key_type{
			DeviceId: config.DeviceId, // 设备id
			PointId:  config.PointId,  // 点位id
		}

		History_Config[key] = config
	}
	return nil
}

/*
******************记录模块******************
 */

type History_func func([]History_Value_type) error

var (
	History_value_list []*History_func
	History_mu         sync.Mutex
)

// 记录模块 发布 发送
func History_Publisher(v []History_Value_type) error {
	if len(v) == 0 {
		return nil
	}

	for _, f := range History_value_list {
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR 记录模块发布失败: %s", err)
		}
	}

	return nil
}

// 记录模块 订阅 接收
func History_Subscriber(value History_func) error {
	History_mu.Lock()
	defer History_mu.Unlock()
	History_value_list = append(History_value_list, &value)
	return nil
}

/*
******************记录业务判断******************
 */

func History_Judgment(new fullConfig.Value_type) (History_Value_type, bool) {
	if new.DeviceId == "" {
		log.Printf("ERROR 获取记录配置失败: DeviceId=='' ")
		return History_Value_type{}, false
	}
	if new.PointId == 0 {
		log.Printf("ERROR 获取记录配置失败: PointId==0 ")
		return History_Value_type{}, false
	}
	key := Config_key_type{
		DeviceId: new.DeviceId, // 设备id
		PointId:  new.PointId,  // 点位id
	}
	cfg, ok := History_Config__Query(key)
	if !ok {
		return History_Value_type{}, false
	}

	if cfg.Config == "" || cfg.Config == "null" {
		return History_Value_type{}, false
	}

	v, ok := byte_util.ConvBool(new.Value, new.Type)
	if !ok {
		err := fmt.Errorf("ERROR modbus_tcp: 读取值类型不匹配, 设备id: %s, 点位id: %d, 配置类型: %s, 值类型: %t", new.DeviceId, new.PointId, new.Type, v)
		log.Print(err)
		return History_Value_type{}, false
	}

	r := History_Value_type{
		DeviceId: new.DeviceId, // 设备id
		PointId:  new.PointId,  // 点位id
		Time:     new.Time,     // 记录时间

		Msg:   new.Msg,   // 状态信息
		Value: new.Value, // 记录值
		Type:  new.Type,  // 值类型
	}

	return r, true
}

func History_Judgment_list(new_list []fullConfig.Value_type) error {
	// 优化：预分配切片容量，避免多次扩容
	History_list := make([]History_Value_type, 0, len(new_list))

	for _, new := range new_list {
		a, ok := History_Judgment(new)
		if !ok {
			continue
		}
		History_list = append(History_list, a)
	}

	// 优化：只在有历史数据时才发布
	if len(History_list) == 0 {
		return nil
	}

	return History_Publisher(History_list)
}

func init() {
	Collection_Subscriber(History_Judgment_list)
}
