/*
* 日期: 2026.6.5 PM3:49
* 作者: 范范zwf
* 作用: 报警数据库
 */

package db_point

import (
	"encoding/json"
	"fmt"
	"log"
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/mysql"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/expr-lang/expr"
)

type Alarm_Config_type struct {
	AlarmId uint      // 报警配置id
	PointId uint      // 点位id
	Config  string    // 配置
	Group   int       // 组
	Status  string    // 状态 开始/结束/触发
	Time    time.Time // 最后一次警告时间
}

var (
	Alarm_Config      map[uint][]Alarm_Config_type // 一个点位可有多条报警
	Alarm_Config_RWMu sync.RWMutex

	// freecache 报警配置缓存
	alarmCache *freecache.Cache
)

func init() {
	Alarm_Config = make(map[uint][]Alarm_Config_type)

	// 初始化 freecache，缓存大小 100MB
	alarmCache = freecache.NewCache(100 * 1024 * 1024)

	// 从配置读取TTL（0=永久保存不过期）
	alarmCacheTTL := int(Init.Config.ALARM.Config_CacheTTL.Seconds())
	if alarmCacheTTL == 0 {
		log.Printf("INFO 报警配置缓存已启用，永久保存（无TTL）")
	} else {
		log.Printf("INFO 报警配置缓存已启用，TTL=%ds", alarmCacheTTL)
	}
}

// alarmCacheKey 生成缓存key（仅用 PointId 区分）
func alarmCacheKey(pointId uint) []byte {
	return fmt.Appendf(nil, "alarm:%d", pointId)
}

// 读取配置（freecache → MySQL → 内存 map）
func Alarm_Config__Query(pointId uint) ([]Alarm_Config_type, bool) {
	cacheKey := alarmCacheKey(pointId)

	// 1. 优先查 freecache
	if data, err := alarmCache.Get(cacheKey); err == nil {
		var cfgs []Alarm_Config_type
		if json.Unmarshal(data, &cfgs) == nil && len(cfgs) > 0 {
			return cfgs, true
		}
	}

	// 2. freecache 未命中，从 MySQL 读取
	mysqlCfgs, err := mysql.Alarm_Config__Query([]uint{pointId}, 0, 0)
	if err == nil && len(mysqlCfgs) > 0 {
		cfgs := make([]Alarm_Config_type, 0, len(mysqlCfgs))
		for _, mc := range mysqlCfgs {
			cfgs = append(cfgs, Alarm_Config_type{
				AlarmId: mc.Id, // 报警配置id
				PointId: mc.Point_Id,
				Config:  mc.Config,
				Group:   mc.Group,
				Status:  "正常",
			})
		}

		// 写入 freecache
		if data, err := json.Marshal(cfgs); err == nil {
			alarmCache.Set(cacheKey, data, int(Init.Config.ALARM.Config_CacheTTL.Seconds()))
		}

		// 同步更新内存 map
		Alarm_Config_RWMu.Lock()
		Alarm_Config[pointId] = cfgs
		Alarm_Config_RWMu.Unlock()

		return cfgs, true
	}

	// 3. MySQL 也未命中，回退查内存 map
	Alarm_Config_RWMu.RLock()
	defer Alarm_Config_RWMu.RUnlock()
	v, ok := Alarm_Config[pointId]
	return v, ok
}

func Alarm_Config__Query_list(keys []uint) (r []Alarm_Config_type) {
	for _, key := range keys {
		cfgs, ok := Alarm_Config__Query(key)
		if ok {
			r = append(r, cfgs...)
		}
	}
	return
}

// 增加配置（同时写入 freecache 和内存 map，按 key 分组追加）
func Alarm_Config__Add(configs []Alarm_Config_type) error {
	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()

	// 按 key 分组，收集所有要更新的 key
	updatedKeys := make(map[uint]struct{})
	for _, config := range configs {
		pointId := config.PointId
		Alarm_Config[pointId] = append(Alarm_Config[pointId], config)
		updatedKeys[pointId] = struct{}{}
	}

	// 同步写入 freecache
	for key := range updatedKeys {
		if data, err := json.Marshal(Alarm_Config[key]); err == nil {
			alarmCache.Set(alarmCacheKey(key), data, int(Init.Config.ALARM.Config_CacheTTL.Seconds()))
		}
	}
	return nil
}

// 修改状态（根据 config 字符串匹配具体哪条报警，同时更新 freecache）
func Alarm_Config__Status_Update(pointId uint, configStr string, status string, t time.Time) error {
	Alarm_Config_RWMu.Lock()
	defer Alarm_Config_RWMu.Unlock()

	cfgs, ok := Alarm_Config[pointId]
	if !ok {
		return fmt.Errorf("tag not found")
	}

	found := false
	for i := range cfgs {
		if cfgs[i].Config == configStr {
			cfgs[i].Status = status
			cfgs[i].Time = t
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("alarm config not found: %s", configStr)
	}

	Alarm_Config[pointId] = cfgs

	// 同步更新 freecache
	if data, err := json.Marshal(cfgs); err == nil {
		alarmCache.Set(alarmCacheKey(pointId), data, int(Init.Config.ALARM.Config_CacheTTL.Seconds()))
	}

	return nil
}

// Alarm_Config__Reload 清空 freecache 和内存 map，从 MySQL 全量重载
// 前端增删改报警配置后调用，确保内存数据与 MySQL 一致
func Alarm_Config__Reload() {
	// 1. 清空 freecache
	alarmCache.Clear()

	// 2. 清空内存 map
	Alarm_Config_RWMu.Lock()
	Alarm_Config = make(map[uint][]Alarm_Config_type)
	Alarm_Config_RWMu.Unlock()

	// 3. 从 MySQL 全量加载
	mysqlCfgs, err := mysql.Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		log.Printf("ERROR 报警配置重载失败: %s", err)
		return
	}

	// 4. 按 PointId 分组写入内存 map 和 freecache
	grouped := make(map[uint][]Alarm_Config_type)
	for _, mc := range mysqlCfgs {
		pointId := mc.Point_Id
		grouped[pointId] = append(grouped[pointId], Alarm_Config_type{
			AlarmId: mc.Id,
			PointId: mc.Point_Id,
			Config:  mc.Config,
			Group:   mc.Group,
			Status:  "正常",
		})
	}

	Alarm_Config_RWMu.Lock()
	Alarm_Config = grouped
	Alarm_Config_RWMu.Unlock()

	ttl := int(Init.Config.ALARM.Config_CacheTTL.Seconds())
	for pointId, cfgs := range grouped {
		if data, err := json.Marshal(cfgs); err == nil {
			alarmCache.Set(alarmCacheKey(pointId), data, ttl)
		}
	}

	log.Printf("INFO 报警配置重载完成，共 %d 条", len(mysqlCfgs))
}

/*
******************报警模块******************
 */

type Alarm_type struct {
	PointId uint      // 点位id
	Time    time.Time // 时间
	Config  string    // 配置
	Group   int       // 组
	Status  string    // 状态 开始/结束/触发

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

// alarmJudgmentSingle 单条报警配置的业务判断
func alarmJudgmentSingle(new fullConfig.Value_type, cfg Alarm_Config_type) (Alarm_type, bool) {
	if cfg.Config == "" || cfg.Config == "null" {
		return Alarm_type{}, false
	}
	if cfg.Group == 0 {
		return Alarm_type{}, false
	}

	// env 用map[string]any承载，x是什么类型直接放进去
	env := map[string]any{
		"x": new.Value,
	}
	// 预编译（实际业务建议把program缓存，不要每次Compile）
	program, compileErr := expr.Compile(cfg.Config, expr.Env(env))

	var alarmVal bool
	var runErr error
	if compileErr != nil {
		log.Printf("ERROR 报警表达式编译失败, config: %s, error: %s", cfg.Config, compileErr)
	} else {
		var res any
		res, runErr = expr.Run(program, env)
		if runErr != nil {
			log.Printf("ERROR 报警表达式执行失败, config: %s, error: %s", cfg.Config, runErr)
		} else {
			alarmVal = byte_util.AnyToBool(res)
		}
	}

	r := Alarm_type{
		PointId: new.PointId, // 点位id
		Time:    new.Time,    // 时间
		Config:  cfg.Config,  // 配置
		Group:   cfg.Group,   // 组
		Status:  "",          // 状态 报警/恢复/错误信息
		Type:    new.Type,    // 点位类型
		Value:   new.Value,   // 点位值
	}
	if compileErr != nil {
		r.Status = compileErr.Error()
	} else if runErr != nil {
		r.Status = runErr.Error()
	} else if alarmVal {
		r.Status = "报警"
	} else {
		r.Status = "恢复"
	}

	return r, true
}

// Alarm_Judgment 遍历该点位的所有报警配置，返回所有触发的报警
func Alarm_Judgment(new fullConfig.Value_type) ([]Alarm_type, bool) {
	if new.PointId == 0 {
		log.Printf("ERROR 获取报警配置失败: PointId==0 ")
		return nil, false
	}

	pointId := new.PointId

	cfgs, ok := Alarm_Config__Query(pointId)
	if !ok {
		return nil, false
	}

	var alarms []Alarm_type
	for _, cfg := range cfgs {
		a, ok := alarmJudgmentSingle(new, cfg)
		if !ok {
			continue
		}
		// 更新状态
		Alarm_Config__Status_Update(pointId, cfg.Config, a.Status, new.Time)
		alarms = append(alarms, a)
	}

	if len(alarms) == 0 {
		return nil, false
	}
	return alarms, true
}

func Alarm_Judgment_list(new_list []fullConfig.Value_type) error {
	// 优化：预分配切片容量，避免多次扩容
	alarm_list := make([]Alarm_type, 0, len(new_list))

	for _, new := range new_list {
		alarms, ok := Alarm_Judgment(new)
		if !ok {
			continue
		}
		alarm_list = append(alarm_list, alarms...)
	}

	// 只在有报警数据时才发布
	if len(alarm_list) == 0 {
		return nil
	}

	return Alarm_Publisher(alarm_list)
}

func init() {
	Update_Subscriber(Alarm_Judgment_list)
	Alarm_Subscriber(func(alarm_list []Alarm_type) error {
		for _, alarm := range alarm_list {
			fmt.Printf("报警触发 点位id: %d, 逻辑: %s, 状态: %s, 值: %v, 时间: %s\n", alarm.PointId, alarm.Config, alarm.Status, alarm.Value, alarm.Time.Format(time.RFC3339))
		}
		return nil
	})
}
