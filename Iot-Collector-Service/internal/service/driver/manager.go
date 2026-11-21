/*
* 日期: 2026-05-31 1:57
* 作者: 范范zwf
* 描述: 驱动管理器 统一管理所有驱动的查询、加载、启动，核心优化：先判断驱动是否支持，再查询点位配置，避免不必要的数据库查询和错误日志
 */

package driver

import (
	"collector/internal/driver/modbus"
	"collector/internal/driver/siemens"
	"collector/internal/model"
	"collector/internal/repo/mysql"

	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"errors"
	"fmt"
	"log"
)

// driverEntry 单个驱动的入口（函数 + 实例）
type driverEntry struct {
	instance any // 驱动实例（如 *Modbus_Tcp.modbus_Tcp）

	// 函数绑定：创建时根据驱动类型赋值，调用时直接执行，无需类型断言
	New          func(mysql.Drive_Config_Query_type, []mysql.Point_Config_Query_type) error
	Connect      func(func([]model.Value_type) error) error
	Close        func() error
	UpdatePoints func([]mysql.Point_Config_Query_type) error // 动态更新点位（不断连接）
	Write        func([]model.Value_type) error              // 写入函数
	callback     func([]model.Value_type) error              // 存储回调，供 ResetConfig 重连时复用
}

// 最关键：驱动管理器（支持 N 个驱动）
type DriverManager struct {
	drivers     map[uint]*driverEntry // key = 驱动ID
	driverTypes map[uint]string       // key = 驱动ID，value = 驱动类型
	driverLock  map[uint]*sync.Mutex  // key = 驱动ID，每驱动独立锁（防止同 id 并发操作）
	mu          sync.RWMutex
}

// Manager 包加载时就绪，任何调用路径下都不为 nil。
// 这点很关键：handler.Web() 与 internalmqtt 比驱动启动更早可用（driver.Start 内部还 sleep 1s 并查库），
// 若只在 InitializeDrivers 里初始化，这个窗口内保存点位/写值/MQTT 指令都会走到 Manager.mu.RLock() 空指针 panic，
// 而 panic 发生在子 goroutine 会直接带走整个进程。
var Manager = newDriverManager()

func newDriverManager() *DriverManager {
	return &DriverManager{
		drivers:     make(map[uint]*driverEntry),
		driverTypes: make(map[uint]string),
		driverLock:  make(map[uint]*sync.Mutex),
	}
}

// InitManager 保留给启动流程调用；幂等：仅在 Manager 为 nil 时重建，
// 绝不把正在轮询的驱动连同各自的锁一起丢掉（旧实现每次调用都会替换整个管理器）。
func InitManager() {
	if Manager == nil {
		Manager = newDriverManager()
	}
}

// CreateDriver 创建驱动实例并存入管理器，根据驱动类型绑定函数
func CreateDriver(driveType string, driveId uint) (any, error) {
	var entry *driverEntry

	switch driveType {
	case "Modbus_Tcp":
		inst := &modbus.Modbus_Tcp{}
		entry = &driverEntry{
			instance:     inst,
			New:          inst.New,
			Connect:      inst.Connect,
			Close:        inst.Close,
			UpdatePoints: inst.UpdatePoints,
			Write:        inst.Write,
		}
	case "Siemens_S7":
		inst := &siemens.Siemens_S7{}
		entry = &driverEntry{
			instance:     inst,
			New:          inst.New,
			Connect:      inst.Connect,
			Close:        inst.Close,
			UpdatePoints: inst.UpdatePoints,
			Write:        inst.Write,
		}
	default:
		return nil, errors.New("不支持的驱动类型: " + driveType)
	}

	Manager.mu.Lock()
	Manager.drivers[driveId] = entry
	Manager.driverTypes[driveId] = driveType
	if _, ok := Manager.driverLock[driveId]; !ok {
		Manager.driverLock[driveId] = &sync.Mutex{}
	}
	Manager.mu.Unlock()

	return entry.instance, nil
}

// GetDriver 根据驱动ID获取驱动实例
func GetDriver(id uint) (any, error) {
	if id == 0 {
		return nil, fmt.Errorf("无效的驱动ID")
	}
	Manager.mu.RLock()
	defer Manager.mu.RUnlock()
	e, ok := Manager.drivers[id]
	if !ok {
		return nil, fmt.Errorf("不存在的驱动 id=%d", id)
	}
	return e.instance, nil
}

// getEntry 获取指定驱动的 entry（内部辅助）
func getEntry(id uint) (*driverEntry, error) {
	Manager.mu.RLock()
	defer Manager.mu.RUnlock()
	e, ok := Manager.drivers[id]
	if !ok {
		return nil, fmt.Errorf("不存在的驱动 id=%d", id)
	}
	return e, nil
}

// getDriverLock 获取指定驱动的独立锁
func getDriverLock(id uint) *sync.Mutex {
	Manager.mu.RLock()
	l, _ := Manager.driverLock[id]
	Manager.mu.RUnlock()
	return l
}

// tryLockDrive 尝试锁定驱动，失败立即返回 error（不阻塞）
func tryLockDrive(id uint) (*sync.Mutex, error) {
	l := getDriverLock(id)
	if l == nil {
		return nil, fmt.Errorf("不存在的驱动 id=%d", id)
	}
	if !l.TryLock() {
		return nil, fmt.Errorf("驱动 id=%d 正在操作中，请稍后再试", id)
	}
	return l, nil
}

// unlockDrive 解锁驱动（操作完成后延迟 3 秒释放）
func unlockDrive(l *sync.Mutex) {
	time.Sleep(3 * time.Second)
	l.Unlock()
}

// IsDriverBusy 检查驱动是否正在操作中（不阻塞，立即返回）
func IsDriverBusy(id uint) bool {
	l := getDriverLock(id)
	if l == nil {
		return false // 驱动不存在，不算忙碌
	}
	if !l.TryLock() {
		return true // 无法获取锁，驱动正在操作中
	}
	l.Unlock() // 获取成功，立即释放
	return false
}

// DriveNew 初始化指定驱动（解析配置、组包等）
func DriveNew(id uint, Drive mysql.Drive_Config_Query_type, Points []mysql.Point_Config_Query_type) error {

	l, err := tryLockDrive(id)
	if err != nil {
		return err
	}
	defer unlockDrive(l)

	e, err := getEntry(id)
	if err != nil {
		return err
	}
	return e.New(Drive, Points)
}

// DriveConnect 连接指定驱动，并绑定外部映射回调
func DriveConnect(id uint, callback func([]model.Value_type) error) error {
	l, err := tryLockDrive(id)
	if err != nil {
		return err
	}
	defer unlockDrive(l)

	e, err := getEntry(id)
	if err != nil {
		return err
	}
	e.callback = callback // 存储回调，供 ResetConfig 复用
	return e.Connect(callback)
}

// DriveClose 关闭指定驱动连接
func DriveClose(id uint) error {
	l, err := tryLockDrive(id)
	if err != nil {
		return err
	}
	defer unlockDrive(l)

	e, err := getEntry(id)
	if err != nil {
		return err
	}
	return e.Close()
}

// DriveResetConfig 重置驱动配置：Close → 6秒后 New → 3秒后 Connect
// 整个过程锁定该 id，同 id 的并发调用立即返回 error
func DriveResetConfig(id uint, Drive mysql.Drive_Config_Query_type, Points []mysql.Point_Config_Query_type) error {
	// 如果驱动在内存中不存在（新添加的），先创建实例
	if _, err := getEntry(id); err != nil {
		if _, err := CreateDriver(Drive.Type, id); err != nil {
			return fmt.Errorf("DriveResetConfig 驱动 id=%d 创建失败: %w", id, err)
		}
	}

	l, err := tryLockDrive(id)
	if err != nil {
		return err
	}
	defer unlockDrive(l)

	e, err := getEntry(id)
	if err != nil {
		return err
	}

	// 1. 先关闭连接
	if err := e.Close(); err != nil {
		log.Printf("WARN DriveResetConfig 驱动 id=%d Close失败: %v", id, err)
	}

	// 2. 等待 6 秒后重新初始化配置
	time.Sleep(6 * time.Second)
	if err := e.New(Drive, Points); err != nil {
		return fmt.Errorf("DriveResetConfig 驱动 id=%d New失败: %w", id, err)
	}

	// 3. 等待 3 秒后重新连接
	time.Sleep(3 * time.Second)
	if err := e.Connect(e.callback); err != nil {
		return fmt.Errorf("DriveResetConfig 驱动 id=%d Connect失败: %w", id, err)
	}

	return nil
}

// DriveUpdatePoints 动态更新指定驱动的点位配置（不断连接，只重组包）
func DriveUpdatePoints(id uint, Points []mysql.Point_Config_Query_type) error {
	e, err := getEntry(id)
	if err != nil {
		return err
	}
	if e.UpdatePoints == nil {
		return fmt.Errorf("驱动 id=%d 不支持动态更新点位", id)
	}
	return e.UpdatePoints(Points)
}

// DriveWrite 写入指定驱动的点位值
// 写值入口统一只读校验：仅 RW_Cancel=3(只写)/4(读写) 允许写入，禁止(1)/只读(2)/不存在的点位整批拒绝，
// 返回携带 Tag 的明确错误。校验放在此集中入口（HTTP 与内部 MQTT 两条通路都汇聚于此），
// 避免依赖各驱动自觉实现而漏校验，也修掉 Modbus 组包静默过滤只读点位后仍假返回成功的问题。
func DriveWrite(id uint, values []model.Value_type) error {
	e, err := getEntry(id)
	if err != nil {
		return err
	}
	if e.Write == nil {
		return fmt.Errorf("驱动 id=%d 不支持写入", id)
	}
	if len(values) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(values))
	for _, v := range values {
		ids = append(ids, v.PointId)
	}
	rwList, err := mysql.Point_Config__Query__RW(ids...)
	if err != nil {
		return fmt.Errorf("查询点位读写属性失败: %w", err)
	}
	rwMap := make(map[uint]int, len(rwList))
	tagMap := make(map[uint]string, len(rwList))
	for _, it := range rwList {
		rwMap[it.Id] = it.RW_Cancel
		tagMap[it.Id] = it.Tag
	}

	denied := make([]string, 0)
	for _, v := range values {
		cancel, exists := rwMap[v.PointId]
		if !exists {
			denied = append(denied, fmt.Sprintf("%s(不存在)", writeTag(v, tagMap)))
			continue
		}
		if cancel != rwCancelWriteOnly && cancel != rwCancelReadWrite {
			denied = append(denied, fmt.Sprintf("%s(RW=%d)", writeTag(v, tagMap), cancel))
		}
	}
	if len(denied) > 0 {
		return fmt.Errorf("%w: 以下点位不允许写入（只读/禁止/不存在）: %s", ErrPointNotWritable, strings.Join(denied, ", "))
	}

	return e.Write(values)
}

// RW_Cancel 可写位常量：1=禁止(N) 2=只读(R) 3=只写(W) 4=读写(R/W)
const (
	rwCancelWriteOnly = 3
	rwCancelReadWrite = 4
)

// ErrPointNotWritable 写值入口校验失败：目标点位为只读/禁止/不存在，请求被拒绝
// （区别于设备写入失败，供上层映射为客户端错误码而非 500）
var ErrPointNotWritable = errors.New("点位不可写")

// writeTag 取点位的展示标识：优先请求携带的 Tag，为空时回退按 id 反查库中的 Tag
func writeTag(v model.Value_type, tagMap map[uint]string) string {
	if v.Tag != "" {
		return v.Tag
	}
	if t := tagMap[v.PointId]; t != "" {
		return t
	}
	return fmt.Sprintf("id=%d", v.PointId)
}

// CloseAll 优雅退出时关闭全部驱动：停止轮询协程、断开设备连接。
// 单驱动最多等 driveCloseWait 拿独占锁，拿不到就跳过（说明正在进行配置操作），
// 整体受 ctx 约束，绝不把退出流程卡死。
func CloseAll(ctx context.Context) {
	Manager.mu.RLock()
	ids := make([]uint, 0, len(Manager.drivers))
	entries := make(map[uint]*driverEntry, len(Manager.drivers))
	for id, e := range Manager.drivers {
		ids = append(ids, id)
		entries[id] = e
	}
	Manager.mu.RUnlock()

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] }) // 日志顺序固定，便于对比

	for _, id := range ids {
		e := entries[id]
		if e == nil || e.Close == nil {
			continue
		}
		if ctx.Err() != nil {
			return
		}

		l := getDriverLock(id)
		if l != nil {
			if !lockWithTimeout(l, driveCloseWait) {
				log.Printf("WARN 驱动 id=%d 正在操作中，退出时不主动关闭（交由进程退出回收）", id)
				continue
			}
		}
		if err := e.Close(); err != nil {
			log.Printf("WARN 驱动 id=%d 关闭失败: %v", id, err)
		} else {
			log.Printf("INFO 驱动 id=%d 连接已关闭", id)
		}
		if l != nil {
			l.Unlock()
		}
	}
}

const driveCloseWait = 2 * time.Second // 退出时等单驱动锁的上限

// lockWithTimeout 在 timeout 内反复 TryLock。用于退出路径：
// 不长期阻塞，又避免与驱动连接/重启操作互踩。
func lockWithTimeout(l *sync.Mutex, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		if l.TryLock() {
			return true
		}
		select {
		case <-deadline:
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
}
