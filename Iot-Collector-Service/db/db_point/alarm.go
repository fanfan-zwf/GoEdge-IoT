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
	"main/IO/conv"
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/mysql"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Alarm_Config_type 报警配置内存结构体，对应 MySQL Alarm_Config 表
// 一个点位可配多条报警规则，运行时动态更新 Status 和 Time
type Alarm_Config_type struct {
	AlarmId uint      // 报警配置 id（对应 MySQL Id）
	PointId uint      // 关联点位 id
	Config  string    // 报警条件表达式（如 "x > 100"）
	Group   int       // 报警组号
	Status  string    // 运行时状态：正常/报警/恢复/错误信息
	Time    time.Time // 最后一次状态变更时间
}

var (
	// Alarm_Config 报警配置内存缓存，key 为 PointId，value 为该点位的所有报警规则
	Alarm_Config map[uint][]Alarm_Config_type
	// Alarm_Config_RWMu 保护 Alarm_Config 的读写锁
	Alarm_Config_RWMu sync.RWMutex

	// alarmCache freecache 报警配置 LRU 缓存，100MB，加速查询
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

// alarmCacheKey 生成 freecache 缓存键
// 功能：将 PointId 格式化为 "alarm:{pointId}" 字节切片
// 传入：pointId — 点位 ID
// 返回：缓存键字节切片
func alarmCacheKey(pointId uint) []byte {
	return fmt.Appendf(nil, "alarm:%d", pointId)
}

// Alarm_Config__Query 单点位报警配置查询
// 功能：三级缓存查询 freecache → MySQL → 内存 map，命中后回填上层缓存
// 传入：pointId — 要查询的点位 ID
// 返回：该点位的报警配置切片，bool 表示是否查到
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

// Alarm_Config__Query_list 批量查询多个点位的报警配置
// 功能：遍历 keys 逐个调用 Alarm_Config__Query，汇总结果
// 传入：keys — 要查询的点位 ID 列表
// 返回：所有命中的报警配置汇总切片
func Alarm_Config__Query_list(keys []uint) (r []Alarm_Config_type) {
	for _, key := range keys {
		cfgs, ok := Alarm_Config__Query(key)
		if ok {
			r = append(r, cfgs...)
		}
	}
	return
}

// Alarm_Config__Add 批量写入报警配置
// 功能：将报警配置追加到内存 map（按 PointId 分组追加），同时同步写入 freecache
// 传入：configs — 要写入的报警配置切片
// 返回：始终返回 nil
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

// Alarm_Config__Status_Update 更新报警配置状态
// 功能：根据 PointId + Config 字符串匹配具体哪条报警规则，更新其 Status 和 Time，同时同步 freecache
// 传入：pointId — 点位 ID；configStr — 报警表达式（用于匹配具体规则）；status — 新状态（报警/恢复/错误信息）；t — 状态变更时间
// 返回：点位不存在或规则未匹配时返回 error，否则返回 nil
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

// Alarm_Config__Reload 报警配置全量重载
// 功能：清空 freecache 和表达式编译缓存 → 从 MySQL 全量加载 → 原子替换内存 map → 同步 freecache
//
//	前端增删改报警配置后调用，确保内存数据与 MySQL 一致
//
// 传入：无
// 返回：无（失败时打印日志并提前返回）
func Alarm_Config__Reload() {
	// 1. 清空 freecache 和表达式编译缓存
	alarmCache.Clear()
	alarmProgramCache = sync.Map{}

	// 2. 从 MySQL 全量加载（不持锁，在内存中构建新数据）
	mysqlCfgs, err := mysql.Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		log.Printf("ERROR 报警配置重载失败: %s", err)
		return
	}

	// 3. 按 PointId 分组
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

	// 4. 一次性原子替换内存 map，避免清空和写入之间的竞态窗口
	Alarm_Config_RWMu.Lock()
	Alarm_Config = grouped
	Alarm_Config_RWMu.Unlock()

	// 5. 同步写入 freecache
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

// Alarm_type 报警事件结构体，报警触发时通过 alarmBus 发布
// 包含报警配置信息、当前点位值和状态（报警/恢复/错误）
type Alarm_type struct {
	Id      uint      // 报警配置 id
	PointId uint      // 触发报警的点位 id
	Time    time.Time // 触发时间
	Config  string    // 触发报警的表达式
	Group   int       // 报警组号
	Status  string    // 状态：报警/恢复/错误信息

	Type  int // 点位值类型
	Value any // 当前点位值
}

// alarmBus 报警事件总线，报警判断触发后通过此总线分发
// 订阅者：init() 中的 MySQL 写入回调
var alarmBus bus[[]Alarm_type]

// Alarm_Publisher 报警事件发布
// 功能：将触发的报警事件分发到 alarmBus 的所有订阅者（如 MySQL Alarm_Data 表写入）
// 传入：v — 本批次触发的报警事件切片
// 返回：nil 或发布过程中的首个错误
func Alarm_Publisher(v []Alarm_type) error {
	if len(v) == 0 {
		return nil
	}
	return alarmBus.Publish(v)
}

// Alarm_Subscriber 注册报警事件订阅者
// 功能：注册回调函数，当报警事件触发时被调用
// 传入：fn — 订阅回调，接收本批次触发的报警事件切片
// 返回：无（内部忽略 Subscribe 返回的 unsubscribe）
func Alarm_Subscriber(fn func([]Alarm_type) error) {
	alarmBus.Subscribe(fn)
}

/*
******************报警业务判断******************
 */

// alarmProgramCache 缓存已编译的表达式程序，key 为报警配置 AlarmId
// 每条报警配置独立缓存，即使表达式相同，不同点位的值类型可能不同，编译结果不能混用
var alarmProgramCache sync.Map // map[uint]*vm.Program

// getOrCompileAlarmProgram 获取或编译报警表达式
// 功能：从 alarmProgramCache 按 AlarmId 查找已编译的 *vm.Program，未命中则编译并缓存
// 传入：alarmId — 报警配置 id（缓存唯一标识）；config — 报警条件表达式字符串（如 "x > 100"）；env — 表达式环境变量（用于类型推断）
// 返回：编译后的 *vm.Program（可并发安全 Run），或编译错误
func getOrCompileAlarmProgram(alarmId uint, config string, env map[string]any) (*vm.Program, error) {
	if prog, ok := alarmProgramCache.Load(alarmId); ok {
		return prog.(*vm.Program), nil
	}
	prog, err := expr.Compile(config, expr.Env(env))
	if err != nil {
		return nil, err
	}
	alarmProgramCache.Store(alarmId, prog)
	return prog, nil
}

// alarmJudgmentSingle 单条报警配置的业务判断
// 功能：用 expr 表达式引擎求值 cfg.Config，x 代表点位值
//
//	表达式结果为 true → 报警，false → 恢复，编译/执行错误 → 错误信息写入 Status
//
// 传入：new — 本条采集的点位值；cfg — 单条报警配置
// 返回：报警事件结构体 + true 表示判断完成；零值 + false 表示配置无效跳过
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
	// 从缓存获取已编译的表达式程序，首次编译后缓存，后续直接复用
	program, compileErr := getOrCompileAlarmProgram(cfg.AlarmId, cfg.Config, env)

	var alarmVal bool
	var runErr error
	if compileErr != nil {
		log.Printf("ERROR 报警【%d】 点位【%d】 表达式编译失败, config: %s, error: %s", cfg.AlarmId, new.PointId, cfg.Config, compileErr)
	} else {
		var res any
		res, runErr = expr.Run(program, env)
		if runErr != nil {
			log.Printf("ERROR 报警【%d】 点位【%d】 表达式执行失败, config: %s, error: %s", cfg.AlarmId, new.PointId, cfg.Config, runErr)
		} else {
			alarmVal = conv.AnyToBool(res)
		}
	}

	r := Alarm_type{
		Id:      cfg.AlarmId, // 报警id
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

// Alarm_Judgment 单点位报警判断
// 功能：查询该点位的所有报警配置，逐条调用 alarmJudgmentSingle 判断表达式，
//
//	每条判断后更新配置状态（Alarm_Config__Status_Update），汇总所有触发的报警
//
// 传入：new — 本条采集的点位值
// 返回：触发的报警事件切片 + true；无报警或 PointId==0 时返回 nil + false
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

// Alarm_Judgment_list 批量报警判断
// 功能：遍历所有点位值，逐点调用 Alarm_Judgment 判断报警，汇总后通过 Alarm_Publisher 发布
// 传入：new_list — 本批次采集的点位值切片
// 返回：始终返回 nil（满足 bus 订阅回调签名）；仅在产生报警数据时才实际发布
func Alarm_Judgment_list(new_list []fullConfig.Value_type) error {
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

// init 报警模块初始化：
// 1. 订阅 updateBus → 值变化时触发报警判断
// 2. 订阅 alarmBus → 报警触发时写入 MySQL Alarm_Data 表
func init() {
	// 值变化 → 报警判断
	updateBus.Subscribe(func(v []fullConfig.Value_type) error {
		if len(v) == 0 {
			return nil
		}

		var a []fullConfig.Value_type
		for _, new := range v {
			if new.PointId == 0 {
				continue
			}

			if new.Value == nil {
				continue
			}

			a = append(a, new)

		}

		return Alarm_Judgment_list(v)
	})

	// 报警触发 → 写入 MySQL
	alarmBus.Subscribe(func(alarm_list []Alarm_type) error {
		var alarmData []mysql.Alarm_Data_Add_type
		for _, alarm := range alarm_list {
			alarmData = append(alarmData, mysql.Alarm_Data_Add_type{
				Alarm_Id: alarm.Id,
				Status:   alarm.Status,
				Time:     alarm.Time,
			})
		}
		if len(alarmData) == 0 {
			return nil
		}
		return mysql.Alarm_Data__Add(alarmData...)
	})
}
