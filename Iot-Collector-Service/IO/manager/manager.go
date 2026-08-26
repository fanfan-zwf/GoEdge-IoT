/*
* 日期: 2026-05-31 1:57
* 作者: 范范zwf
* 描述: 驱动管理器 统一管理所有驱动的查询、加载、启动，核心优化：先判断驱动是否支持，再查询点位配置，避免不必要的数据库查询和错误日志
 */

package manager

import (
	"main/IO/Modbus_Tcp"
	"main/IO/manager/fullConfig"
	"main/db/mysql"

	"sync"
	"time"

	"errors"
	"fmt"
	"log"
)

// driverEntry 单个驱动的入口（函数 + 实例）
type driverEntry struct {
	instance any // 驱动实例（如 *Modbus_Tcp.Modbus_Tcp）

	// 函数绑定：创建时根据驱动类型赋值，调用时直接执行，无需类型断言
	New      func(mysql.Drive_Config_type, []mysql.CollectorGet_Point_Config_type) error
	Connect  func(func([]fullConfig.Value_type) error) error
	Close    func() error
	callback func([]fullConfig.Value_type) error // 存储回调，供 ResetConfig 重连时复用
}

// 最关键：驱动管理器（支持 N 个驱动）
type DriverManager struct {
	drivers     map[uint]*driverEntry // key = 驱动ID
	driverTypes map[uint]string       // key = 驱动ID，value = 驱动类型
	driverLock  map[uint]*sync.Mutex  // key = 驱动ID，每驱动独立锁（防止同 id 并发操作）
	mu          sync.RWMutex
}

var Manager *DriverManager

func InitManager() {
	Manager = &DriverManager{
		drivers:     make(map[uint]*driverEntry),
		driverTypes: make(map[uint]string),
		driverLock:  make(map[uint]*sync.Mutex),
	}
}

// CreateDriver 创建驱动实例并存入管理器，根据驱动类型绑定函数
func CreateDriver(driveType string, driveId uint) (any, error) {
	var entry *driverEntry

	switch driveType {
	case "Modbus_Tcp":
		inst := &Modbus_Tcp.Modbus_Tcp{}
		entry = &driverEntry{
			instance: inst,
			New:      inst.New,
			Connect:  inst.Connect,
			Close:    inst.Close,
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

// DriveNew 初始化指定驱动（解析配置、组包等）
func DriveNew(id uint, Drive mysql.Drive_Config_type, Points []mysql.CollectorGet_Point_Config_type) error {

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
func DriveConnect(id uint, callback func([]fullConfig.Value_type) error) error {
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
func DriveResetConfig(id uint, Drive mysql.Drive_Config_type, Points []mysql.CollectorGet_Point_Config_type) error {
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
