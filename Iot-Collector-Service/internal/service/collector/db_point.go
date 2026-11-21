/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据核心层——IO 采集数据分发、值变化检测、实时值内存存储
*       通过泛型 bus[T] 发布/订阅总线连接各模块，数据流：
*       IO 驱动 → collectionBus → 值变化检测 → updateBus → 报警判断
*                                → 历史记录判断
 */

package collector

import (
	"collector/internal/model"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

/*
******************实时值存储******************
* 内存中维护所有点位的最新值，供 WebSocket 推送、API 查询等使用
* 通过 Value_Map_Mu（RWMutex）保护并发读写
 */

var (
	// Value_Map 点位实时值映射，key 为 PointId
	Value_Map map[uint]Update_Value_type
	// Value_Map_Mu 保护 Value_Map 的读写锁
	Value_Map_Mu sync.RWMutex
)

func init() {
	Value_Map = make(map[uint]Update_Value_type)
}

// Update_Value_type 点位实时值，嵌入采集值并记录上一次的值和时间
// 用于值变化检测：对比新旧值决定是否触发 Update 总线
type Update_Value_type struct {
	model.Value_type

	Last_Value any       // 上一次的点位值
	Last_Time  time.Time // 上一次值变化的时间
}

// Value_Map__Query_list 实时值批量查询
// 功能：根据点位 ID 列表从内存中批量查询实时值（当前值 + 上一次值）
// 传入：keys — 要查询的点位 ID 列表
// 返回：匹配的实时值切片，不存在的 ID 会被跳过
func Value_Map__Query_list(keys []uint) (r []Update_Value_type) {
	Value_Map_Mu.RLock()
	defer Value_Map_Mu.RUnlock()
	for _, pointId := range keys {
		v, ok := Value_Map[pointId]
		if ok {
			r = append(r, v)
		}
	}
	return
}

/*
******************泛型发布/订阅总线******************
 */

// bus 泛型发布/订阅总线，替代重复的 pub/sub 模板代码
// 解决原实现中 Publisher 遍历未加锁、无 Unsubscribe 等问题
type bus[T any] struct {
	mu          sync.RWMutex
	subscribers []func(T) error
}

// Subscribe 注册订阅者
// 功能：将回调函数追加到订阅列表，支持并发安全
// 传入：fn — 订阅回调，接收泛型数据 T，返回 error 表示处理结果
// 返回：unsubscribe — 取消订阅函数，调用后该订阅者将被置空不再触发
func (b *bus[T]) Subscribe(fn func(T) error) (unsubscribe func()) {
	b.mu.Lock()
	idx := len(b.subscribers)
	b.subscribers = append(b.subscribers, fn)
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if idx < len(b.subscribers) {
			b.subscribers[idx] = nil // 置空，Publish 时跳过
		}
	}
}

// Publish 发布数据到所有订阅者
// 功能：先 RLock 拷贝订阅列表快照，再逐个调用回调，避免并发 append 导致 panic
// 传入：v — 要发布的数据（泛型 T）
// 返回：始终返回 nil；单个订阅者报错或 panic 时仅打印日志，不影响其他订阅者
func (b *bus[T]) Publish(v T) error {
	b.mu.RLock()
	subs := make([]func(T) error, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, fn := range subs {
		if fn == nil {
			continue
		}

		// 单个订阅者 panic 不得影响其他订阅者，更不能让进程退出：
		// Publish 常由裸 goroutine 调用（如 Update_Publisher），recover 是唯一可用的兜底边界
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("ERROR 订阅者回调 panic: %v\n%s", r, debug.Stack())
				}
			}()

			if err := fn(v); err != nil {
				log.Printf("ERROR 发布失败: %s", err)
			}
		}()
	}
	return nil
}

/*
******************IO 数据采集总线******************
 */

// collectionBus IO 采集数据总线，IO 驱动采集到数据后通过此总线分发
// 订阅者：Update_Value_Judgment_list（值变化检测）、History_Judgment_list（历史记录）
var collectionBus bus[[]model.Value_type]

// Collection_Publisher IO 数据采集发布
// 功能：将 IO 驱动层采集到的原始点位数据分发到 collectionBus 的所有订阅者（值变化检测、历史记录判断）
// 传入：v — 本批次采集到的点位值切片
// 返回：nil 或发布过程中的首个错误
func Collection_Publisher(v []model.Value_type) error {
	if len(v) == 0 {
		return nil
	}
	return collectionBus.Publish(v)
}

// Collection_Subscriber 注册 IO 数据采集订阅者
// 功能：注册回调函数，当 IO 驱动采集到数据时被调用
// 传入：fn — 订阅回调，接收本批次采集的点位值切片
// 返回：无（内部忽略 Subscribe 返回的 unsubscribe）
func Collection_Subscriber(fn func([]model.Value_type) error) {
	collectionBus.Subscribe(fn)
}

/*
******************数据更新总线******************
 */

// updateBus 数据变化总线，仅在点位值发生变化时发布
// 订阅者：Alarm_Judgment_list（报警判断）、influxdb（时序写入）、WebSocket（前端推送）
var updateBus bus[[]model.Value_type]

// Update_Publisher 数据变化发布
// 功能：将发生变化的点位值分发到 updateBus 的所有订阅者（报警判断、InfluxDB 写入、WebSocket 推送）
// 传入：v — 本批次发生值变化的点位值切片
// 返回：nil 或发布过程中的首个错误
func Update_Publisher(v []model.Value_type) error {
	if len(v) == 0 {
		return nil
	}
	return updateBus.Publish(v)
}

// Update_Subscriber 注册数据变化订阅者
// 功能：注册回调函数，当点位值发生变化时被调用（供 InfluxDB、WebSocket 等外部模块使用）
// 传入：fn — 订阅回调，接收本批次变化的点位值切片
// 返回：无（内部忽略 Subscribe 返回的 unsubscribe）
func Update_Subscriber(fn func([]model.Value_type) error) {
	updateBus.Subscribe(fn)
}

/*
******************值变化检测******************
 */

// Update_Value_Judgment_single 单条数据变化检测（内部函数）
// 功能：对比新采集值与内存中的旧值，判断是否发生变化；新点位直接入库视为变化
// 传入：new — 本条采集的点位值（需包含 PointId、Value、Time、Type、Msg）
// 返回：true 表示值有变化（已更新内存），false 表示值未变化
// 注意：调用方必须持有 Value_Map_Mu 写锁
func Update_Value_Judgment_single(new model.Value_type) bool {
	key := new.PointId

	old, ok := Value_Map[key]

	if !ok {
		// 新点位：完整初始化所有字段
		Value_Map[key] = Update_Value_type{
			Value_type: model.Value_type{
				PointId: new.PointId,
				Tag:     new.Tag, // 点位标识（历史数据按此索引）
				Time:    new.Time,
				Value:   new.Value,
				Type:    new.Type,
				Msg:     new.Msg,
			},
			Last_Value: new.Value,
			Last_Time:  new.Time,
		}
		return true
	}

	// 值变化检测
	if new.Value != old.Value || new.Msg != old.Msg {
		old.Last_Value = old.Value
		old.Last_Time = old.Time
		old.Value = new.Value
		old.Time = new.Time
		old.Msg = new.Msg
		Value_Map[key] = old
		return true
	}

	return false
}

// Update_Value_Judgment_list 批量数据变化检测
// 功能：一次写锁覆盖整个批次，逐条调用 Update_Value_Judgment_single 检测变化，
//
//	收集所有变化的点位后在锁外异步发布到 updateBus
//
// 传入：new_list — 本批次采集的点位值切片
// 返回：始终返回 nil（满足 bus 订阅回调签名）
func Update_Value_Judgment_list(new_list []model.Value_type) error {
	if len(new_list) == 0 {
		return nil
	}

	Value_Map_Mu.Lock()
	update_list := make([]model.Value_type, 0, len(new_list))
	for _, v := range new_list {
		if Update_Value_Judgment_single(v) {
			update_list = append(update_list, v)
		}
	}
	Value_Map_Mu.Unlock()

	// 锁外发布，避免持锁调用外部回调
	// Publish 内部已逐个 recover 且恒返回 nil，此处无需处理返回值
	if len(update_list) > 0 {
		go func() { _ = Update_Publisher(update_list) }()
	}

	return nil
}

/*
******************初始化 & 数据流接线******************
 */

func init() {
	// IO 采集 → 值变化检测
	collectionBus.Subscribe(Update_Value_Judgment_list)
}

// New 模块初始化入口
// 功能：db_point 模块的对外初始化接口，当前无需额外操作，预留扩展
// 传入：无
// 返回：始终返回 nil
func New() error {
	return nil
}
