/*
* 日期: 2026.7.5 PM5:54
* 作者: 范范zwf
* 作用: 设备状态管理
 */

package db_point

import (
	"sync"
	"time"
)

// Drive_Status_type 设备状态数据结构
type Drive_Status_type struct {
	Id             uint      // 设备ID
	Last_Push_Time time.Time // 最后一次推送时间
	Status         string    // 设备状态
}

// Drive_Status 设备状态管理器
type Drive_Status struct {
	Status_Callback func([]Drive_Status_type) error // 状态变化回调函数

	status_map map[uint]Drive_Status_type // 设备状态映射表
	status_mu  sync.RWMutex               // 读写锁，保护并发访问
}

// Status_New 创建设备状态管理器实例
func Status_New() *Drive_Status {
	return &Drive_Status{
		status_map: make(map[uint]Drive_Status_type),
	}
}

// Update 批量更新设备状态
// 参数 status: 可变参数，支持一次更新多个设备状态
// 注意: 回调函数在锁外执行，避免长时间持有锁影响并发性能
// Update 批量更新设备状态
// 参数 status: 可变参数，支持一次更新多个设备状态
// 注意: 回调函数在锁外执行，避免长时间持有锁影响并发性能
func (c *Drive_Status) Update(status ...Drive_Status_type) {
	// 如果没有传入状态，直接返回
	if len(status) == 0 {
		return
	}

	// 预分配切片容量，避免动态扩容带来的性能开销
	vals := make([]Drive_Status_type, 0, len(status))

	// 加锁处理数据更新
	c.status_mu.Lock()
	defer c.status_mu.Unlock()

	for _, statu := range status {
		s, exists := c.status_map[statu.Id]

		// 总是更新时间戳，保证最后推送时间最新
		s.Last_Push_Time = statu.Last_Push_Time

		// 状态不一致时才更新
		if s.Status != statu.Status {
			s.Status = statu.Status
		}

		// 首次出现或状态为空字符串时不触发回调
		if exists && statu.Status != "" && statu.Status != s.Status {
			vals = append(vals, s)
		}

		// 更新到状态映射表中
		c.status_map[statu.Id] = s
	}

	// 在锁外执行回调，避免长时间持有锁影响并发性能
	if len(vals) > 0 && c.Status_Callback != nil {
		c.Status_Callback(vals)
	}
}

func (c *Drive_Status) GetStatus(id uint) (Drive_Status_type, bool) {
	c.status_mu.RLock()
	defer c.status_mu.RUnlock()
	statu, exists := c.status_map[id]
	return statu, exists
}
