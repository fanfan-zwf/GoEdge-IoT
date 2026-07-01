/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"log"
	"main/IO/manager/fullConfig"
	"time"

	"fmt"
	"sync"
)

var (
	Err_Publisher_Close = fmt.Errorf("Close")
	Value_Map           map[string]Update_Value_type
	Value_Map_Mu        sync.RWMutex
)

func init() {
	Value_Map = make(map[string]Update_Value_type)
}

/*
******************IO数据采集******************
 */

type Collection_func func([]fullConfig.Value_type) error

var (
	Collection_value_list []*Collection_func
	Collection_mu         sync.Mutex
)

// IO数据采集 发布 发送
func Collection_Publisher(v []fullConfig.Value_type) error {
	if len(v) == 0 {
		return nil
	}

	for _, f := range Collection_value_list {
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR IO数据采集发布失败: %s", err)

		}
	}

	return nil
}

// IO数据采集 订阅 接收
func Collection_Subscriber(value Collection_func) error {
	Collection_mu.Lock()
	defer Collection_mu.Unlock()
	Collection_value_list = append(Collection_value_list, &value)
	return nil
}

/*
******************数据更新******************
 */

// 数据更新结构体
type Update_Value_type struct {
	fullConfig.Value_type

	Last_Value any // 值
	Last_Time  time.Time
}

type Update_func func([]fullConfig.Value_type) error

var (
	Update_Value_list []*Update_func
	Update_mu         sync.Mutex
)

// 数据更新 发布 发送
func Update_Publisher(v []fullConfig.Value_type) error {
	if len(v) == 0 {
		return nil
	}

	for _, f := range Update_Value_list {
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR 数据更新发布失败: %s", err)
		}
	}

	return nil
}

// 数据更新 订阅 接收
func Update_Subscriber(value Update_func) error {
	Update_mu.Lock()
	defer Update_mu.Unlock()
	Update_Value_list = append(Update_Value_list, &value)
	return nil
}

// 单条数据更新判断
func Update_Value_Judgment(new fullConfig.Value_type) (bool, error) {
	// 优化：使用写锁保护整个读写过程，避免竞态窗口
	Value_Map_Mu.Lock()
	defer Value_Map_Mu.Unlock()
	
	old, ok := Value_Map[new.Tag]
	
	if !ok {
		// 修复Bug: 新点位需要完整初始化所有字段
		Value_Map[new.Tag] = Update_Value_type{
			Value_type: fullConfig.Value_type{
				Tag:   new.Tag,
				Time:  new.Time,
				Value: new.Value,
				Type:  new.Type,
				Msg:   new.Msg,
			},
			Last_Value: new.Value, // 首次更新时，Last_Value 与当前值相同
			Last_Time:  new.Time,  // 首次更新时，Last_Time 与当前时间相同
		}
		return true, nil
	}
	
	// 状态消息更新
	if new.Msg != "ok" {
		old.Msg = new.Msg
	}

	// 值变化检测
	if new.Value != old.Value {
		old.Last_Value = old.Value
		old.Last_Time = old.Time
		old.Value = new.Value
		old.Time = new.Time
		
		// 在锁内直接更新，避免二次加锁
		Value_Map[new.Tag] = old
		return true, nil
	}
	
	// 值未变化
	return false, nil
}

// 批量数据更新判断
func Update_Value_Judgment_list(new_list []fullConfig.Value_type) error {
	if len(new_list) == 0 {
		return nil
	}

	// 优化：预分配切片容量，避免多次扩容
	update_list := make([]fullConfig.Value_type, 0, len(new_list))
	
	for _, v := range new_list {
		u, err := Update_Value_Judgment(v)
		if err != nil {
			log.Printf("ERROR 数据更新判断失败: %s", err)
			// 修复：继续处理其他数据，不因单个错误中断整个批次
			continue
		}
		if u {
			update_list = append(update_list, v)
		}
	}
	
	// 优化：只在有变化数据时才发布
	if len(update_list) > 0 {
		Update_Publisher(update_list)
	}
	
	return nil
}
func init() {
	Collection_Subscriber(Update_Value_Judgment_list)
}

// 初始化
func New() error {
	return nil
}

func init() {
	Update_Subscriber(func(v []fullConfig.Value_type) error {
		for _, val := range v {
			fmt.Printf("数据更新 Tag: %s, Value: %v, Msg: %s, Time: %s\n", val.Tag, val.Value, val.Msg, val.Time.Format(time.RFC3339))
		}
		return nil
	})
}
