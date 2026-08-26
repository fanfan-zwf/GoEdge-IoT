/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"log"
	"main/IO/manager/fullConfig"

	"reflect"
	"sync"
)

/*
******************写入******************
 */

type Write_value_func_type func([]fullConfig.Value_type) (err error)

var (
	Write_value    map[Config_key_type]*Write_value_func_type // 存储点位和写入函数的关系
	Write_value_mu sync.Mutex
)

func init() {
	Write_value = make(map[Config_key_type]*Write_value_func_type)
}

// 变化更新 发布 发送
func Write_value_Publisher(values []fullConfig.Value_type) error {
	if len(values) == 0 {
		return nil
	}

	// 通过 Write_value 查找回调，根据函数指针判断是否同一个回调，聚合相同回调的点位值
	Write_value_mu.Lock()
	type group struct {
		fn     *Write_value_func_type
		values []fullConfig.Value_type
	}
	groups := make(map[uintptr]*group)
	var order []uintptr

	for _, v := range values {
		key := Config_key_type{DeviceId: v.DeviceId, PointId: v.PointId}
		fn, ok := Write_value[key]
		if !ok {
			continue
		}
		ptr := reflect.ValueOf(*fn).Pointer()
		g, exists := groups[ptr]
		if exists {
			g.values = append(g.values, v)
		} else {
			groups[ptr] = &group{fn: fn, values: []fullConfig.Value_type{v}}
			order = append(order, ptr)
		}
	}
	Write_value_mu.Unlock()

	// 逐回调写入，相同回调的点位值已聚合
	for _, ptr := range order {
		g := groups[ptr]
		if err := (*g.fn)(g.values); err != nil {
			return err
		}
	}

	return nil
}

// 变化更新 订阅 接收
func Write_value_Subscriber(keys []Config_key_type, value Write_value_func_type) error {
	Write_value_mu.Lock()
	defer Write_value_mu.Unlock()

	for _, key := range keys {
		_, ok := Write_value[key]
		if !ok {
			Write_value[key] = &value
		} else {
			log.Printf("ERROR 重复点位 设备id:%s, 点位id:%d", key.DeviceId, key.PointId)
		}
	}

	return nil
}
