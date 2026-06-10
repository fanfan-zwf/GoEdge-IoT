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
func Update_Publisher(v []fullConfig.Value_type) (err error) {
	if len(v) == 0 {
		return nil
	}

	for _, f := range Update_Value_list {
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR 数据更新发布失败: %s", err)
		}
	}

	return
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
	Value_Map_Mu.RLock()
	old, ok := Value_Map[new.Tag]
	Value_Map_Mu.RUnlock()

	if !ok {
		old = Update_Value_type{
			Value_type: fullConfig.Value_type{
				Tag:   new.Tag,
				Time:  new.Time,
				Value: new.Value,
			},
		}
	} else {
		if new.Msg != "ok" {
			old.Msg = new.Msg
		}

		if new.Value != old.Value {
			old.Last_Value = old.Value
			old.Last_Time = old.Time
			old.Value = new.Value
			old.Time = new.Time
		} else {
			return false, nil
		}

	}

	Value_Map_Mu.Lock()
	Value_Map[new.Tag] = old
	Value_Map_Mu.Unlock()
	return true, nil
}

// 批量数据更新判断
func Update_Value_Judgment_list(new_list []fullConfig.Value_type) error {
	if len(new_list) == 0 {
		return nil
	}

	var update_list []fullConfig.Value_type
	for _, v := range new_list {
		u, err := Update_Value_Judgment(v)
		if err != nil {
			log.Printf("ERROR 数据更新判断失败: %s", err)
		}
		if u {
			update_list = append(update_list, v)
		}
	}
	Update_Publisher(update_list)
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
