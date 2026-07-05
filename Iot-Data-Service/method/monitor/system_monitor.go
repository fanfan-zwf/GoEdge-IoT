package monitor

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/db_point"

	"github.com/shirou/gopsutil/v3/process"
)

// SystemMonitor Go 程序资源监控器（内存分配、Goroutine数量、GC次数）
type SystemMonitor struct {
	interval time.Duration // 监控间隔
	stopChan chan struct{}
}

// NewSystemMonitor 创建系统监控器
func NewSystemMonitor(interval time.Duration) *SystemMonitor {
	if interval <= 0 {
		interval = 5 * time.Minute // 默认5分钟
	}
	return &SystemMonitor{
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 启动监控
func (sm *SystemMonitor) Start() {
	go sm.monitorLoop()
}

// Stop 停止监控
func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
}

// monitorLoop 监控循环（后台定时发布程序资源统计信息）
func (sm *SystemMonitor) monitorLoop() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db_point.Collection_Publisher(sm.GetStats()) // 发布程序资源统计信息
		case <-sm.stopChan:
			return
		}
	}
}

// GetStats 获取当前 Go 程序资源统计信息（按照 Value_type 格式返回）
func (sm *SystemMonitor) GetStats() []fullConfig.Value_type {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	now := time.Now()

	// 获取当前进程的 CPU 使用率
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	cpuPercent := 0.0
	if err == nil {
		percent, _ := proc.Percent(0) // 获取瞬时 CPU 使用率
		cpuPercent = percent
	}

	pointPrefix := Init.Config.Monitor.Point

	// 按照 Value_type 格式返回 Go 程序资源使用情况
	return []fullConfig.Value_type{
		// === CPU 相关 ===
		{
			Tag:   fmt.Sprintf("%s/CPU使用率", pointPrefix), // CPU 使用率（%）
			Value: cpuPercent,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},

		// === 内存相关 ===
		{
			Tag:   fmt.Sprintf("%s/当前分配的堆内存MB", pointPrefix), // 当前分配的堆内存（MB）
			Value: float64(memStats.Alloc) / 1024 / 1024,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/累计分配的堆内存MB", pointPrefix), // 累计分配的堆内存（MB）
			Value: float64(memStats.TotalAlloc) / 1024 / 1024,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/从系统获取的内存MB", pointPrefix), // 从系统获取的内存（MB）
			Value: float64(memStats.Sys) / 1024 / 1024,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/堆内存分配MB", pointPrefix), // 堆内存分配（MB）
			Value: float64(memStats.HeapAlloc) / 1024 / 1024,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/栈内存使用MB", pointPrefix), // 栈内存使用（MB）
			Value: float64(memStats.StackInuse) / 1024 / 1024,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},

		// === GC 相关 ===
		{
			Tag:   fmt.Sprintf("%s/GC执行次数", pointPrefix), // GC 执行次数
			Value: memStats.NumGC,
			Type:  "int",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/GC总暂停时间ms", pointPrefix), // GC 总暂停时间（毫秒）
			Value: float64(memStats.PauseTotalNs) / 1e6,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/GC上次暂停时间ms", pointPrefix), // 上次 GC 暂停时间（毫秒）
			Value: float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e6,
			Type:  "float",
			Msg:   "ok",
			Time:  now,
		},

		// === 并发相关 ===
		{
			Tag:   fmt.Sprintf("%s/Goroutine数量", pointPrefix), // Goroutine 数量
			Value: runtime.NumGoroutine(),
			Type:  "int",
			Msg:   "ok",
			Time:  now,
		},

		// === 内存分配统计 ===
		{
			Tag:   fmt.Sprintf("%s/累计内存分配次数", pointPrefix), // 累计内存分配次数
			Value: memStats.Mallocs,
			Type:  "uint64",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/累计内存释放次数", pointPrefix), // 累计内存释放次数
			Value: memStats.Frees,
			Type:  "uint64",
			Msg:   "ok",
			Time:  now,
		},
		{
			Tag:   fmt.Sprintf("%s/指针查找次数", pointPrefix), // 指针查找次数
			Value: memStats.Lookups,
			Type:  "uint64",
			Msg:   "ok",
			Time:  now,
		},
	}
}
