/*
* 日期: 2026.6.5 PM3:49
* 作者: 范范zwf
* 作用: 报警数据库
 */

package db_point

import (
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"

	"errors"
	"fmt"
	"log"

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
func Alarm_Config__Query_list(tag []string) (r []Alarm_Config_type) {
	for _, v := range tag {
		v, ok := Alarm_Config__Query(v)
		if ok {
			r = append(r, v)
		}
	}
	return
}

// 增加配置
func Alarm_Config__Add(configs []Alarm_Config_type) error {
	// 优化：批量操作，只在循环外加一次锁，避免N次锁开销
	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()
	
	for _, config := range configs {
		Alarm_Config[config.Tag] = config
	}
	return nil
}

// 修改状态
func Alarm_Config__Status_Update(tag string, status string, t time.Time) error {
	// 修复Bug: 修改数据必须使用写锁 Lock()，而不是读锁 RLock()
	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()
	
	v, ok := Alarm_Config[tag]
	if !ok {
		return errors.New("tag not found")
	}
	v.Status = status
	v.Time = t
	Alarm_Config[tag] = v
	return nil
}

// 变化更新 订阅 接收 mysql配置
func Alarm_Config_Subscriber_mysqlconfig(cfg fullConfig.FullConfig_type) error {
	// 优化：先收集所有需要更新的配置，然后批量加锁更新
	updates := make(map[string]Alarm_Config_type)
	
	for _, v := range cfg.Points {
		if v.Alarm == "" || v.Alarm == "null" {
			continue
		}

		if v.Alarm_Group == 0 {
			continue
		}

		// 安全判断
		if v.Tag == "" {
			continue
		}

		// 先在局部变量中构建数据
		updates[v.Tag] = Alarm_Config_type{
			Tag:    v.Tag,         // 点位标签
			Config: v.Alarm,       // 配置
			Group:  v.Alarm_Group, // 组
			Status: "正常",
		}
	}
	
	// 优化：批量更新，只加一次锁
	if len(updates) > 0 {
		Alarm_Config_RWMu.Lock()
		for tag, config := range updates {
			Alarm_Config[tag] = config
		}
		Alarm_Config_RWMu.Unlock()
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
	Alarm_mu         sync.Mutex
)

// 报警模块 发布 发送
func Alarm_Publisher(v []Alarm_type) error {
	if len(v) == 0 {
		return nil
	}

	for _, f := range Alarm_value_list {
		err := (*f)(v)
		if err != nil {
			log.Printf("ERROR 报警模块发布失败: %s", err)
		}
	}

	return nil
}

// 报警模块 订阅 接收
func Alarm_Subscriber(value Alarm_func) error {
	Alarm_mu.Lock()
	defer Alarm_mu.Unlock()
	Alarm_value_list = append(Alarm_value_list, &value)
	return nil
}

/*
******************报警业务判断******************
 */

func Alarm_Judgment(new fullConfig.Value_type) (Alarm_type, bool) {
	if new.Tag == "" {
		log.Printf("ERROR 获取报警配置失败: Tag=='' ")
		return Alarm_type{}, false
	}

	cfg, ok := Alarm_Config__Query(new.Tag)
	if !ok {
		return Alarm_type{}, false
	}

	if cfg.Config == "" || cfg.Config == "null" {
		return Alarm_type{}, false
	}

	if cfg.Group == 0 {
		return Alarm_type{}, false
	}

	v, ok := byte_util.ConvBool(new.Value, new.Type)
	if !ok {
		err := fmt.Errorf("ERROR modbus_tcp: 读取值类型不匹配, tag: %s, 配置类型: %s, 值类型: %t", new.Tag, new.Type, v)
		log.Print(err)
		return Alarm_type{}, false
	}

	r := Alarm_type{
		Tag:    new.Tag,    // 点位名称
		Time:   new.Time,   // 时间
		Config: cfg.Config, // 配置
		Group:  cfg.Group,  // 组
		Status: " ",        // 状态 开始/结束/触发
		Type:   new.Type,   // 点位类型
		Value:  new.Value,  // 点位值
	}
	if cfg.Config == "==0" && !v {
		r.Status = "开始"
	} else if cfg.Config == "==0" && v {
		r.Status = "恢复"
	} else if cfg.Config == "==1" && v {
		r.Status = "开始"
	} else if cfg.Config == "==1" && !v {
		r.Status = "恢复"
	} else if cfg.Config == "0<>1" {
		r.Status = "触发"
	} else {
		return Alarm_type{}, false
	}

	Alarm_Config__Status_Update(new.Tag, r.Status, new.Time)

	return r, true
}

func Alarm_Judgment_list(new_list []fullConfig.Value_type) error {
	// 优化：预分配切片容量，避免多次扩容
	alarm_list := make([]Alarm_type, 0, len(new_list))
	
	for _, new := range new_list {
		a, ok := Alarm_Judgment(new)
		if !ok {
			continue
		}
		alarm_list = append(alarm_list, a)
	}
	
	// 只在有报警数据时才发布
	if len(alarm_list) == 0 {
		return nil
	}
	
	return Alarm_Publisher(alarm_list)
}

func init() {
	Update_Subscriber(Alarm_Judgment_list)
}
