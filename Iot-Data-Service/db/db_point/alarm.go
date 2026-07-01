/*
* 日期: 2026.6.5 PM3:49
* 作者: 范范zwf
* 作用: 报警数据库
 */

package db_point

import (
	"errors"
	"fmt"
	"log"
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"sync"
	"time"
)

type Alarm_Config_type struct {
	Tag    string    // 点位标签
	Config string    // 配置
	Group  int       // 组
	Status string    // 状态 开始/结束/触发
	Time   time.Time // 最后一次警告时间
}

var (
	Alarm_Config      map[string]Alarm_Config_type
	Alarm_Config_RWMu sync.RWMutex
)

func init() {
	Alarm_Config = make(map[string]Alarm_Config_type)
}

// 读取配置
func Alarm_Config__Query(tag string) (Alarm_Config_type, bool) {
	Alarm_Config_RWMu.RLock()
	defer Alarm_Config_RWMu.RUnlock()
	v, ok := Alarm_Config[tag]
	return v, ok
}

func Alarm_Config__Query_list(tags []string) []Alarm_Config_type {
	if len(tags) == 0 {
		return nil
	}

	// 预分配容量
	result := make([]Alarm_Config_type, 0, len(tags))

	Alarm_Config_RWMu.RLock()
	defer Alarm_Config_RWMu.RUnlock()

	for _, tag := range tags {
		if v, ok := Alarm_Config[tag]; ok {
			result = append(result, v)
		}
	}
	return result
}

// 增加配置（批量优化版）
func Alarm_Config__Add(configs []Alarm_Config_type) error {
	if len(configs) == 0 {
		return nil
	}

	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()

	for _, config := range configs {
		// 安全校验
		if config.Tag == "" {
			log.Printf("WARN 跳过无效报警配置: Tag为空")
			continue
		}
		Alarm_Config[config.Tag] = config
	}
	return nil
}

// 修改状态（修复数据竞争 Bug）
func Alarm_Config__Status_Update(tag string, status string, t time.Time) error {
	if tag == "" {
		return errors.New("tag不能为空")
	}

	Alarm_Config_RWMu.Lock() // ✅ 修复：使用写锁
	defer Alarm_Config_RWMu.Unlock()

	v, ok := Alarm_Config[tag]
	if !ok {
		return errors.New("tag not found")
	}

	v.Status = status
	v.Time = t
	Alarm_Config[tag] = v // ✅ 在写锁保护下安全写入
	return nil
}

// 变化更新 订阅 接收 mysql配置
func Alarm_Config_Subscriber_mysqlconfig(cfg fullConfig.FullConfig_type) error {
	if len(cfg.Points) == 0 {
		return nil
	}

	// 先收集需要更新的配置
	updates := make(map[string]Alarm_Config_type)

	for _, v := range cfg.Points {
		// 安全判断
		if v.Tag == "" || v.Alarm == "" || v.Alarm == "null" || v.Alarm_Group == 0 {
			continue
		}

		updates[v.Tag] = Alarm_Config_type{
			Tag:    v.Tag,         // 点位标签
			Config: v.Alarm,       // 配置
			Group:  v.Alarm_Group, // 组
			Status: "正常",
			Time:   time.Now(),
		}
	}

	// 批量写入（只加一次锁）
	if len(updates) > 0 {
		Alarm_Config_RWMu.Lock()
		defer Alarm_Config_RWMu.Unlock() // ✅ 修复：使用 defer 确保锁一定会被释放
		
		for tag, config := range updates {
			Alarm_Config[tag] = config
		}
	}

	return nil
}

/*
******************报警模块******************
 */

type Alarm_type struct {
	Tag    string    // 点位名称
	Time   time.Time // 时间
	Config string    // 配置
	Group  int       // 组
	Status string    // 状态 开始/结束/触发

	Type  string // 点位类型
	Value any    // 点位值
}

type Alarm_func func([]Alarm_type) error

var (
	Alarm_value_list []*Alarm_func
	Alarm_mu         sync.RWMutex // ✅ 改为 RWMutex
)

// 报警模块 发布 发送
func Alarm_Publisher(v []Alarm_type) error {
	if len(v) == 0 {
		return nil
	}

	Alarm_mu.RLock() // ✅ 使用读锁
	defer Alarm_mu.RUnlock()

	for _, f := range Alarm_value_list {
		if f == nil { // ✅ 空指针检查
			continue
		}
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR 报警模块发布失败: %s", err)
		}
	}

	return nil
}

// 报警模块 订阅 接收
func Alarm_Subscriber(value Alarm_func) error {
	if value == nil {
		return errors.New("订阅函数不能为nil")
	}

	Alarm_mu.Lock()
	defer Alarm_mu.Unlock()

	// 避免重复订阅
	for _, existing := range Alarm_value_list {
		if existing == &value {
			return nil
		}
	}

	Alarm_value_list = append(Alarm_value_list, &value)
	return nil
}

/*
******************报警业务判断******************
 */

// 判断报警状态（优化版）
func Alarm_Judgment(new fullConfig.Value_type) (Alarm_type, bool) {
	// 参数校验
	if new.Tag == "" {
		log.Printf("ERROR 报警判断失败: Tag为空")
		return Alarm_type{}, false
	}

	cfg, ok := Alarm_Config__Query(new.Tag)
	if !ok {
		return Alarm_type{}, false
	}

	if cfg.Config == "" || cfg.Config == "null" || cfg.Group == 0 {
		return Alarm_type{}, false
	}

	// 类型转换
	v, ok := byte_util.ConvBool(new.Value, new.Type)
	if !ok {
		err := fmt.Errorf("报警判断: 值类型转换失败, tag=%s, 值=%v, 类型=%s",
			new.Tag, new.Value, new.Type)
		log.Print(err)
		return Alarm_type{}, false
	}

	r := Alarm_type{
		Tag:    new.Tag,
		Time:   new.Time,
		Config: cfg.Config,
		Group:  cfg.Group,
		Status: "", // 初始化为空
		Type:   new.Type,
		Value:  new.Value,
	}

	// 根据配置判断报警状态
	switch cfg.Config {
	case "==0":
		if !v {
			r.Status = "开始"
		} else {
			r.Status = "恢复"
		}
	case "==1":
		if v {
			r.Status = "开始"
		} else {
			r.Status = "恢复"
		}
	case "0<>1":
		r.Status = "触发"
	default:
		return Alarm_type{}, false
	}

	// 异步更新状态（避免阻塞主流程）
	go func(tag, status string, t time.Time) {
		if err := Alarm_Config__Status_Update(tag, status, t); err != nil {
			log.Printf("WARN 更新报警状态失败 tag=%s: %v", tag, err)
		}
	}(new.Tag, r.Status, new.Time)

	return r, true
}

// 批量报警判断（修复变量遮蔽 + 性能优化）
func Alarm_Judgment_list(new_list []fullConfig.Value_type) error {
	if len(new_list) == 0 {
		return nil
	}

	// 预分配容量
	alarm_list := make([]Alarm_type, 0, len(new_list))

	for _, newVal := range new_list { // ✅ 修复：改用 newVal 避免遮蔽内置函数
		a, ok := Alarm_Judgment(newVal)
		if !ok {
			continue
		}
		alarm_list = append(alarm_list, a)
	}

	// 只有存在报警时才发布
	if len(alarm_list) > 0 {
		if err := Alarm_Publisher(alarm_list); err != nil {
			log.Printf("ERROR 报警发布失败: %v", err)
			return err
		}
	}

	return nil
}

// 获取所有报警配置快照（新增功能）
func GetAllAlarmConfigs() map[string]Alarm_Config_type {
	Alarm_Config_RWMu.RLock()
	defer Alarm_Config_RWMu.RUnlock()

	result := make(map[string]Alarm_Config_type, len(Alarm_Config))
	for k, v := range Alarm_Config {
		result[k] = v
	}
	return result
}

// 删除报警配置（新增功能）
func RemoveAlarmConfig(tag string) {
	if tag == "" {
		return
	}

	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()
	delete(Alarm_Config, tag)
}

func init() {
	Update_Subscriber(Alarm_Judgment_list)
}
