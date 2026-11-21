package internalmqtt

import (
	"collector/internal/config"
	"collector/internal/model"
	"collector/internal/repo/mysql"
	"collector/internal/service/collector"
	"collector/pkg/conv"
	"collector/pkg/sysinfo"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// ======== 点位 tag 缓存 ========
// Value_Map 中的实时值不含 Tag，发布时用本缓存按 PointId 补全标签；
// 写入方向则按 tag 反查点位 id 与所属驱动。启动时加载，全量推送与配置更新后刷新。

// pointRef 由 tag 反查得到的点位定位信息。
type pointRef struct {
	id      uint
	driveId uint
}

var (
	tagMap     map[uint]string     // PointId → Tag（发布补全）
	pointByTag map[string]pointRef // Tag → {id, driveId}（写入定位）
	tagMapMu   sync.RWMutex
)

// refreshTagMap 从 MySQL 重新加载 全点位的 id→tag 与 tag→{id,driveId} 双向映射。
func refreshTagMap() {
	points, err := mysql.Point_Config__Query(nil, 0, 0)
	if err != nil {
		log.Printf("WARN 内部MQTT刷新tag缓存失败: %v", err)
		return
	}
	idTag := make(map[uint]string, len(points))
	byTag := make(map[string]pointRef, len(points))
	for _, p := range points {
		idTag[p.Id] = p.Tag
		byTag[p.Tag] = pointRef{id: p.Id, driveId: p.Drive_Id}
	}
	tagMapMu.Lock()
	tagMap = idTag
	pointByTag = byTag
	tagMapMu.Unlock()
}

// tagOf 读取某点位的标签，缓存未命中返回空串。
func tagOf(pointId uint) string {
	tagMapMu.RLock()
	defer tagMapMu.RUnlock()
	if tagMap == nil {
		return ""
	}
	return tagMap[pointId]
}

// lookupPoint 按 tag 反查点位 id 与驱动 id；未命中返回 ok=false。
func lookupPoint(tag string) (pointRef, bool) {
	tagMapMu.RLock()
	defer tagMapMu.RUnlock()
	ref, ok := pointByTag[tag]
	return ref, ok
}

// toDataItems 将采集值切片转换为对外发布格式（以 tag 为键，补全类型名）。
func toDataItems(values []model.Value_type) []dataItem {
	items := make([]dataItem, 0, len(values))
	for _, v := range values {
		items = append(items, dataItem{
			Tag:   tagOf(v.PointId),
			Value: v.Value,
			Type:  conv.ValueTypeIntToString(v.Type),
			Msg:   v.Msg,
			Time:  v.Time,
		})
	}
	return items
}

// ======== 读数据发布（/data） ========

// startDataPublisher 订阅值变化流即时推送，并启动全量周期推送协程。
func startDataPublisher() {
	refreshTagMap()

	// 值变化即时推送
	collector.Update_Subscriber(func(values []model.Value_type) error {
		if len(values) == 0 {
			return nil
		}
		b, err := json.Marshal(toDataItems(values))
		if err != nil {
			return nil
		}
		return publish(cfg.pubData, b)
	})

	// 全量周期推送
	go func() {
		interval := config.Config.Internal_Mqtt.DataFullInterval
		if interval <= 0 {
			interval = 10 * time.Minute
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				publishAllData()
			}
		}
	}()
}

// publishAllData 推送 Value_Map 中所有点位的当前值（并顺带刷新 tag 缓存）。
func publishAllData() {
	refreshTagMap()

	collector.Value_Map_Mu.RLock()
	values := make([]model.Value_type, 0, len(collector.Value_Map))
	for _, uv := range collector.Value_Map {
		values = append(values, uv.Value_type)
	}
	collector.Value_Map_Mu.RUnlock()

	if len(values) == 0 {
		return
	}
	b, err := json.Marshal(toDataItems(values))
	if err != nil {
		return
	}
	_ = publish(cfg.pubData, b)
}

// ======== 报警事件发布（/alarm） ========

// startAlarmPublisher 订阅报警事件总线：每当一批报警触发/恢复，将报警配置字段+报警值+时间组装成数组即时发布。
func startAlarmPublisher() {
	collector.Alarm_Subscriber(func(alarms []collector.Alarm_type) error {
		if len(alarms) == 0 {
			return nil
		}
		items := make([]alarmItem, 0, len(alarms))
		for _, a := range alarms {
			items = append(items, alarmItem{
				Tag:     tagOf(a.PointId),
				AlarmId: a.Id,
				Config:  a.Config,
				Group:   a.Group,
				Status:  a.Status,
				Value:   a.Value,
				Type:    conv.ValueTypeIntToString(a.Type),
				Time:    a.Time,
			})
		}
		b, err := json.Marshal(items)
		if err != nil {
			return nil
		}
		return publish(cfg.pubAlarm, b)
	})
}

// ======== 当前监控状态发布（/status/<功能>） ========

// statusCollector 单个监控功能：子主题名 + 采集函数。
type statusCollector struct {
	name string
	fn   func(context.Context) (any, error)
}

// statusCollectors 对外发布的监控功能清单，每个功能对应一个子主题（如 /status/cpu/percent）。
// 需要增减监控项时，仅在此处增删条目即可。
func statusCollectors() []statusCollector {
	return []statusCollector{
		{"host", func(c context.Context) (any, error) { return sysinfo.CollectHostInfo(c) }},
		{"sensors", func(c context.Context) (any, error) { return sysinfo.CollectSensors(c) }},

		{"cpu", func(c context.Context) (any, error) { return sysinfo.CollectCPUInfo(c) }},
		{"cpu/percent", func(c context.Context) (any, error) {
			return sysinfo.CollectCPUPercent(c, sysinfo.DefaultCPUInterval, false)
		}},
		{"cpu/cores", func(c context.Context) (any, error) { return sysinfo.CollectCPUCounts(c, true) }},

		{"mem", func(c context.Context) (any, error) { return sysinfo.CollectVirtualMemory(c) }},
		{"mem/swap", func(c context.Context) (any, error) { return sysinfo.CollectSwapMemory(c) }},

		{"disk", func(c context.Context) (any, error) { return sysinfo.CollectDiskPartitions(c, false) }},
		{"disk/usage", func(c context.Context) (any, error) { return sysinfo.CollectDiskUsage(c, "/") }},
		{"disk/io", func(c context.Context) (any, error) { return sysinfo.CollectDiskIOCounters(c) }},

		{"load", func(c context.Context) (any, error) { return sysinfo.CollectLoadAvg(c) }},

		{"net", func(c context.Context) (any, error) { return sysinfo.CollectNetIOCounters(c, true) }},
		{"net/interfaces", func(c context.Context) (any, error) { return sysinfo.CollectNetInterfaces(c) }},

		{"process/pids", func(c context.Context) (any, error) { return sysinfo.CollectPids(c) }},
	}
}

// startStatusPublisher 启动监控状态周期发布协程：每个采集周期逐功能采集并发布到各自子主题。
func startStatusPublisher() {
	go func() {
		interval := config.Config.Internal_Mqtt.StatusInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				publishStatusAll()
			}
		}
	}()
}

// publishStatusAll 依次采集并发布全部监控功能；单功能失败仅记日志，不影响其余功能。
func publishStatusAll() {
	list := statusCollectors()
	for _, sc := range list {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		data, err := sc.fn(ctx)
		cancel()
		if err != nil {
			log.Printf("WARN 内部MQTT监控采集[%s]失败: %v", sc.name, err)
			continue
		}
		b, merr := json.Marshal(data)
		if merr != nil {
			log.Printf("WARN 内部MQTT监控序列化[%s]失败: %v", sc.name, merr)
			continue
		}
		_ = publish(joinTopic(cfg.pubStatusPrefix, sc.name), b)
	}
}
