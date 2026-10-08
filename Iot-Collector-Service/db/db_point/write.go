/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 点位写入回调注册与聚合下发
 */

package db_point

import (
	"log"
	"main/IO/manager/fullConfig"
	"sync"
	"sync/atomic"
)

/*
******************写入******************
 */

// Write_value_func_type 写入回调函数类型
// 接收一批点位值，执行写入操作（如 Modbus 写寄存器），返回 error 表示写入结果
type Write_value_func_type func([]fullConfig.Value_type) error

// writeEntry 写入条目，绑定回调函数和分组 ID
type writeEntry struct {
	fn      Write_value_func_type
	groupId uint
}

var (
	Write_value    map[uint]writeEntry // PointId → 写入条目
	Write_value_mu sync.RWMutex
	writeGroupSeq  atomic.Uint64 // 分组 ID 自增序列
)

func init() {
	Write_value = make(map[uint]writeEntry)
}

// Write_value_Publisher 写入回调聚合下发
// 功能：按 groupId 聚合点位值，同一批次注册的点位（相同 groupId）的值会被聚合后
//
//	一次性调用回调，减少写入次数。锁内分组，锁外逐组调用回调
//
// 传入：values — 本批次待写入的点位值切片
// 返回：始终返回 nil
func Write_value_Publisher(values []fullConfig.Value_type) error {
	if len(values) == 0 {
		return nil
	}

	// 按 groupId 聚合
	Write_value_mu.RLock()
	type group struct {
		fn     Write_value_func_type
		values []fullConfig.Value_type
	}
	groups := make(map[uint]*group)
	var order []uint

	for _, v := range values {
		entry, ok := Write_value[v.PointId]
		if !ok {
			continue
		}
		g, exists := groups[entry.groupId]
		if exists {
			g.values = append(g.values, v)
		} else {
			groups[entry.groupId] = &group{fn: entry.fn, values: []fullConfig.Value_type{v}}
			order = append(order, entry.groupId)
		}
	}
	Write_value_mu.RUnlock()

	// 逐组写入，相同回调的点位值已聚合
	for _, gid := range order {
		g := groups[gid]
		if err := g.fn(g.values); err != nil {
			log.Printf("ERROR 写入失败, group=%d: %s", gid, err)
		}
	}

	return nil
}

// Write_value_Subscriber 注册点位写入回调
// 功能：为一批点位注册写入回调，同一批 keys 共享同一个 groupId，
//
//	Publisher 时会聚合为一次回调调用。重复注册的点位会打印警告并跳过
//
// 传入：keys — 要注册回调的点位 ID 列表；value — 写入回调函数
// 返回：始终返回 nil
func Write_value_Subscriber(keys []uint, value Write_value_func_type) error {
	Write_value_mu.Lock()
	defer Write_value_mu.Unlock()

	gid := uint(writeGroupSeq.Add(1))

	for _, pointId := range keys {
		if _, ok := Write_value[pointId]; ok {
			log.Printf("WARN 重复注册写入回调, 点位 id:%d", pointId)
			continue
		}
		Write_value[pointId] = writeEntry{fn: value, groupId: gid}
	}

	return nil
}
