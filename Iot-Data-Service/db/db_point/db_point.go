/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"encoding/json"
	"fmt"
	"log"
	"main/IO/manager/fullConfig"
	"sync"
	"time"
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
	Collection_mu         sync.RWMutex // 改为 RWMutex，读多写少场景
)

// IO数据采集 发布 发送
func Collection_Publisher(v []fullConfig.Value_type) error {
	if len(v) == 0 {
		return nil
	}

	Collection_mu.RLock() // 使用读锁，允许多个发布者并发读取
	defer Collection_mu.RUnlock()

	for _, f := range Collection_value_list {
		if f == nil {
			continue
		}
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

	// 检查是否已存在相同的订阅者（可选优化）
	for _, existing := range Collection_value_list {
		if existing == &value {
			return nil // 避免重复订阅
		}
	}

	Collection_value_list = append(Collection_value_list, &value)
	return nil
}

/*
******************数据更新******************
 */

// 数据更新结构体
type Update_Value_type struct {
	fullConfig.Value_type

	Last_Value any       // 上次的值
	Last_Time  time.Time // 上次的时间
}

type Update_func func([]fullConfig.Value_type) error

var (
	Update_Value_list []*Update_func
	Update_mu         sync.RWMutex // 改为 RWMutex，读多写少场景
)

// 数据更新 发布 发送
func Update_Publisher(v []fullConfig.Value_type) error {
	if len(v) == 0 {
		return nil
	}

	Update_mu.RLock() // 使用读锁，允许多个发布者并发读取
	defer Update_mu.RUnlock()

	for _, f := range Update_Value_list {
		if f == nil {
			continue
		}
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

	// 检查是否已存在相同的订阅者（可选优化）
	for _, existing := range Update_Value_list {
		if existing == &value {
			return nil // 避免重复订阅
		}
	}

	Update_Value_list = append(Update_Value_list, &value)
	return nil
}

// 单条数据更新判断
func Update_Value_Judgment(new fullConfig.Value_type) (bool, error) {
	Value_Map_Mu.Lock() // 直接使用写锁，避免 RLock->Lock 的升级问题
	defer Value_Map_Mu.Unlock()

	old, ok := Value_Map[new.Tag]

	if !ok {
		// 新点位，直接存储
		Value_Map[new.Tag] = Update_Value_type{
			Value_type: fullConfig.Value_type{
				Tag:   new.Tag,
				Time:  new.Time,
				Value: new.Value,
				Type:  new.Type,
				Msg:   new.Msg,
			},
			Last_Value: nil,
			Last_Time:  time.Time{},
		}
		return true, nil
	}

	// 已存在的点位
	changed := false

	// 更新状态信息（无论值是否变化）
	if new.Msg != "" && new.Msg != old.Msg {
		old.Msg = new.Msg
		changed = true
	}

	// 检查值是否变化
	if new.Value != old.Value {
		old.Last_Value = old.Value
		old.Last_Time = old.Time
		old.Value = new.Value
		old.Time = new.Time
		changed = true
	}

	if changed {
		Value_Map[new.Tag] = old
		return true, nil
	}

	return false, nil
}

// 批量数据更新判断
func Update_Value_Judgment_list(new_list []fullConfig.Value_type) error {
	if len(new_list) == 0 {
		return nil
	}

	// 预分配切片容量，减少内存重新分配
	update_list := make([]fullConfig.Value_type, 0, len(new_list))

	for _, v := range new_list {
		u, err := Update_Value_Judgment(v)
		if err != nil {
			log.Printf("ERROR 数据更新判断失败 Tag=%s: %s", v.Tag, err)
			continue // 出错时跳过该条目，继续处理其他数据
		}
		if u {
			update_list = append(update_list, v)
		}
	}

	// 只有当有变化的数据时才发布
	if len(update_list) > 0 {
		if err := Update_Publisher(update_list); err != nil {
			log.Printf("ERROR 批量数据更新发布失败: %s", err)
			return err
		}
	}

	return nil
}

// 获取点位当前值（新增功能）
func GetValue(tag ...string) []Update_Value_type {
	var result []Update_Value_type
	GetValueFlow(tag, func(val Update_Value_type) {
		result = append(result, val)
	})
	return result
}

func GetValueFlow(tag []string, callback func(Update_Value_type)) {
	Value_Map_Mu.RLock()
	defer Value_Map_Mu.RUnlock()

	for _, t := range tag {
		val, ok := Value_Map[t]
		if !ok {
			continue
		}
		callback(val)
	}
}

// 获取所有点位当前值（新增功能）
func GetAllValuesFlow(callback func(Update_Value_type)) {
	Value_Map_Mu.RLock()
	defer Value_Map_Mu.RUnlock()

	for k, v := range Value_Map {
		if k != v.Tag { // 确保Tag一致性
			continue
		}
		callback(v)
	}
}

// 删除点位（新增功能）
func RemoveValue(tag ...string) {
	Value_Map_Mu.Lock()
	defer Value_Map_Mu.Unlock()
	for _, t := range tag {
		delete(Value_Map, t)
	}
}

func init() {
	// 注册默认订阅者：将采集的数据进行更新判断
	Collection_Subscriber(Update_Value_Judgment_list)
}

// 初始化
func New() error {
	return nil
}

// parseTypedValue 根据类型解析JSON值（提取公共逻辑，减少代码重复）
func parseTypedValue(value json.RawMessage, typeStr string) (any, error) {
	switch typeStr {
	case "bool":
		var v bool
		return v, json.Unmarshal(value, &v)
	case "int8":
		var v int8
		return v, json.Unmarshal(value, &v)
	case "uint8":
		var v uint8
		return v, json.Unmarshal(value, &v)
	case "int16":
		var v int16
		return v, json.Unmarshal(value, &v)
	case "uint16":
		var v uint16
		return v, json.Unmarshal(value, &v)
	case "int32":
		var v int32
		return v, json.Unmarshal(value, &v)
	case "uint32":
		var v uint32
		return v, json.Unmarshal(value, &v)
	case "int64":
		var v int64
		return v, json.Unmarshal(value, &v)
	case "uint64":
		var v uint64
		return v, json.Unmarshal(value, &v)
	case "int":
		var v int
		return v, json.Unmarshal(value, &v)
	case "uint":
		var v uint
		return v, json.Unmarshal(value, &v)
	case "float32":
		var v float32
		return v, json.Unmarshal(value, &v)
	case "float64":
		var v float64
		return v, json.Unmarshal(value, &v)
	case "string":
		var v string
		return v, json.Unmarshal(value, &v)
	default:
		return nil, fmt.Errorf("不支持的类型: %s", typeStr)
	}
}

// Json_struct_to_list 将JSON字符串转换为fullConfig.Value_type切片
func Json_struct_to_list(json_str string) ([]fullConfig.Value_type, error) {
	var json_list []struct {
		Tag   string          `json:"tag"`
		Value json.RawMessage `json:"value"`
		Type  string          `json:"type"`
		Msg   string          `json:"msg"`
		Time  time.Time       `json:"time"`
	}

	// 第一步：解析JSON列表
	err := json.Unmarshal([]byte(json_str), &json_list)
	if err != nil {
		return nil, fmt.Errorf("JSON列表解析失败: %w", err)
	}

	// 优化：预分配切片容量，避免多次扩容（遵循"Go 变量初始化冗余优化经验"）
	list := make([]fullConfig.Value_type, 0, len(json_list))

	for _, item := range json_list {
		// 验证Tag有效性（遵循"Go 代码健壮性检查规范"）
		if item.Tag == "" {
			log.Printf("跳过空Tag的数据项")
			continue
		}

		// 使用提取的辅助函数解析类型化值
		realValue, parseErr := parseTypedValue(item.Value, item.Type)
		if parseErr != nil {
			log.Printf("解析值失败 Tag=%s Type=%s: %s", item.Tag, item.Type, parseErr)
			continue // 遵循"数据过滤机制"：跳过无效数据
		}

		list = append(list, fullConfig.Value_type{
			Tag:   item.Tag,
			Value: realValue,
			Type:  item.Type,
			Msg:   item.Msg,
			Time:  item.Time,
		})
	}

	return list, nil
}

// Json_struct_to 将JSON字符串转换为单个fullConfig.Value_type
func Json_struct_to(json_str string) (fullConfig.Value_type, error) {
	var item struct {
		Tag   string          `json:"tag"`
		Value json.RawMessage `json:"value"`
		Type  string          `json:"type"`
		Msg   string          `json:"msg"`
		Time  time.Time       `json:"time"`
	}

	// 第一步：解析JSON结构
	err := json.Unmarshal([]byte(json_str), &item)
	if err != nil {
		return fullConfig.Value_type{}, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 第二步：验证Tag有效性（遵循"Go 代码健壮性检查规范"）
	if item.Tag == "" {
		return fullConfig.Value_type{}, fmt.Errorf("Tag不能为空")
	}

	// 第三步：使用提取的辅助函数解析类型化值
	realValue, parseErr := parseTypedValue(item.Value, item.Type)
	if parseErr != nil {
		log.Printf("解析值失败 Tag=%s Type=%s: %s", item.Tag, item.Type, parseErr)
		return fullConfig.Value_type{}, fmt.Errorf("解析值失败 Tag=%s Type=%s: %w", item.Tag, item.Type, parseErr)
	}

	return fullConfig.Value_type{
		Tag:   item.Tag,
		Value: realValue,
		Type:  item.Type,
		Msg:   item.Msg,
		Time:  item.Time,
	}, nil
}
