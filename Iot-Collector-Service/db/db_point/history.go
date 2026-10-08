/*
* 日期: 2026.6.5 PM3:49
* 作者: 范范zwf
* 作用: 记录数据库
 */

package db_point

import (
	"log"
	"main/IO/conv"
	"main/IO/manager/fullConfig"
	"main/db/mysql"
	"sync"
	"time"
)

// History_Value_type 历史记录值，记录点位的采集数据快照
// 通过 historyBus 发布给下游存储系统（如 InfluxDB）
type History_Value_type struct {
	PointId uint      // 点位 id
	Time    time.Time // 记录时间

	Msg   string // 状态信息（如 "ok" 或错误描述）
	Value any    // 记录值（原始采集值）
	Type  int    // 值类型（类型编码，参见 conv convertType.go）
}

// History_Config_type 历史配置，决定某个点位是否启用历史记录
// Config 非空且非 "null" 时启用，否则不记录
type History_Config_type struct {
	PointId uint   // 点位 id
	Config  string // 配置（内容无关，仅判断是否启用）
}

var (
	// History_Config 历史配置内存缓存，key 为 PointId
	History_Config map[uint]History_Config_type
	// History_Config_RWMu 保护 History_Config 的读写锁
	History_Config_RWMu sync.RWMutex
)

func init() {
	History_Config = make(map[uint]History_Config_type)
}

// History_Config__Query 查询单个点位的历史配置
// 功能：从内存 map 单级缓存读取点位的历史配置
// 传入：pointId — 要查询的点位 ID
// 返回：历史配置结构体 + true 表示找到；零值 + false 表示未找到
func History_Config__Query(pointId uint) (History_Config_type, bool) {
	History_Config_RWMu.RLock()
	defer History_Config_RWMu.RUnlock()
	v, ok := History_Config[pointId]
	return v, ok
}

// History_Config__Query_list 批量查询多个点位的历史配置
// 功能：遍历 keys 逐个调用 History_Config__Query，汇总结果
// 传入：keys — 要查询的点位 ID 列表
// 返回：所有命中的历史配置汇总切片
func History_Config__Query_list(keys []uint) (r []History_Config_type) {
	for _, pointId := range keys {
		v, ok := History_Config__Query(pointId)
		if ok {
			r = append(r, v)
		}
	}
	return
}

// History_Config__Add 批量写入历史配置
// 功能：将历史配置写入内存 map，一次性加锁覆盖整个批次
// 传入：configs — 要写入的历史配置切片
// 返回：始终返回 nil
func History_Config__Add(configs []History_Config_type) error {
	History_Config_RWMu.Lock()
	defer History_Config_RWMu.Unlock()

	for _, config := range configs {
		History_Config[config.PointId] = config
	}
	return nil
}

// History_Config__Reload 历史配置全量重载
// 功能：从 MySQL 全量加载历史配置 → 在内存中构建新 map → 原子替换内存 map
//
//	前端增删改历史配置后调用，确保内存数据与 MySQL 一致
//
// 传入：无
// 返回：无（失败时打印日志并提前返回）
func History_Config__Reload() {
	// 从 MySQL 全量加载
	mysqlCfgs, err := mysql.History_Config__Query(nil, 0, 0)
	if err != nil {
		log.Printf("ERROR 历史配置重载失败: %s", err)
		return
	}

	// 按 PointId 构建新 map
	newMap := make(map[uint]History_Config_type, len(mysqlCfgs))
	for _, mc := range mysqlCfgs {
		newMap[mc.Point_Id] = History_Config_type{
			PointId: mc.Point_Id,
			Config:  mc.Config,
		}
	}

	// 一次性原子替换
	History_Config_RWMu.Lock()
	History_Config = newMap
	History_Config_RWMu.Unlock()

	log.Printf("INFO 历史配置重载完成，共 %d 条", len(mysqlCfgs))
}

/*
******************记录模块******************
 */

// historyBus 历史记录事件总线，历史判断触发后通过此总线分发
// 订阅者：下游存储系统（如 InfluxDB）
var historyBus bus[[]History_Value_type]

// History_Publisher 历史记录发布
// 功能：将待记录的历史数据分发到 historyBus 的所有订阅者（如 InfluxDB 时序写入）
// 传入：v — 本批次待记录的历史数据切片
// 返回：nil 或发布过程中的首个错误
func History_Publisher(v []History_Value_type) error {
	if len(v) == 0 {
		return nil
	}
	return historyBus.Publish(v)
}

// History_Subscriber 注册历史记录订阅者
// 功能：注册回调函数，当历史数据产生时被调用
// 传入：fn — 订阅回调，接收本批次待记录的历史数据切片
// 返回：无（内部忽略 Subscribe 返回的 unsubscribe）
func History_Subscriber(fn func([]History_Value_type) error) {
	historyBus.Subscribe(fn)
}

/*
******************记录业务判断******************
 */

// History_Judgment 单点位历史判断
// 功能：校验配置是否启用（Config 非空且非 "null"）→ 类型校验（ConvBool 检查值与声明类型是否匹配）→ 构造记录值
// 传入：new — 本条采集的点位值
// 返回：历史记录值 + true 表示需要记录；零值 + false 表示不记录（未启用/类型不匹配/PointId==0）
func History_Judgment(new fullConfig.Value_type) (History_Value_type, bool) {
	if new.PointId == 0 {
		log.Printf("ERROR 获取记录配置失败: PointId==0 ")
		return History_Value_type{}, false
	}
	cfg, ok := History_Config__Query(new.PointId)
	if !ok {
		return History_Value_type{}, false
	}

	if cfg.Config == "" || cfg.Config == "null" {
		return History_Value_type{}, false
	}

	// 类型校验：检查值是否与声明的类型匹配
	if _, ok := conv.ConvBool(new.Value, new.Type); !ok {
		log.Printf("ERROR 历史记录类型不匹配, 点位id: %d, 期望类型: %d", new.PointId, new.Type)
		return History_Value_type{}, false
	}

	r := History_Value_type{
		PointId: new.PointId, // 点位id
		Time:    new.Time,    // 记录时间

		Msg:   new.Msg,   // 状态信息
		Value: new.Value, // 记录值
		Type:  new.Type,  // 值类型
	}

	return r, true
}

// History_Judgment_list 批量历史判断
// 功能：遍历所有点位值，逐点调用 History_Judgment 判断是否记录，汇总后通过 History_Publisher 发布
// 传入：new_list — 本批次采集的点位值切片
// 返回：始终返回 nil（满足 bus 订阅回调签名）；仅在产生有效历史数据时才实际发布
func History_Judgment_list(new_list []fullConfig.Value_type) error {
	History_list := make([]History_Value_type, 0, len(new_list))

	for _, new := range new_list {
		a, ok := History_Judgment(new)
		if !ok {
			continue
		}
		History_list = append(History_list, a)
	}

	if len(History_list) == 0 {
		return nil
	}

	return History_Publisher(History_list)
}

func init() {
	// IO 采集 → 历史记录判断
	collectionBus.Subscribe(History_Judgment_list)
}
