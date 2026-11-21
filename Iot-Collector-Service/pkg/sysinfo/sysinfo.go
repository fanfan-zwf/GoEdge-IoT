/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 系统信息采集包，统一封装 github.com/shirou/gopsutil 支持的全部模块。
*       设计原则：gopsutil 有多少个采集功能，就提供多少个独立函数，
*       每个函数直接返回该功能的原始结果（值类型优先、不做拼接、不聚合），
*       既可供对外 HTTP 接口按功能逐一路由复用，也可供监控平台按需单独采集。
 */
package sysinfo

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// DefaultCPUInterval CPU 使用率默认采样间隔。
const DefaultCPUInterval = time.Second

// defaultProtoCounters 网络协议计数器默认统计的协议列表。
var defaultProtoCounters = []string{"tcp", "udp", "udp6", "ip", "icmp", "icmp6"}

// ================= CPU =================

// CollectCPUInfo 采集 CPU 型号等静态信息，直接返回 gopsutil 原始切片。
func CollectCPUInfo(ctx context.Context) ([]cpu.InfoStat, error) {
	return cpu.InfoWithContext(ctx)
}

// CollectCPUCounts 采集 CPU 核心数；logical=true 返回逻辑核心数，false 返回物理核心数。
func CollectCPUCounts(ctx context.Context, logical bool) (int, error) {
	return cpu.CountsWithContext(ctx, logical)
}

// CollectCPUPercent 采集 CPU 使用率百分比；interval 为采样间隔（<=0 不阻塞返回启动以来均值），
// perCPU=true 按核拆分，false 返回单条总体使用率。
func CollectCPUPercent(ctx context.Context, interval time.Duration, perCPU bool) ([]float64, error) {
	return cpu.PercentWithContext(ctx, interval, perCPU)
}

// CollectCPUTimes 采集 CPU 时间统计（user/system/idle 等累计值）；perCPU=true 按核拆分。
func CollectCPUTimes(ctx context.Context, perCPU bool) ([]cpu.TimesStat, error) {
	return cpu.TimesWithContext(ctx, perCPU)
}

// ================= Mem =================

// CollectVirtualMemory 采集物理内存使用情况，返回 gopsutil 原始结构（值类型）。
func CollectVirtualMemory(ctx context.Context) (mem.VirtualMemoryStat, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || v == nil {
		return mem.VirtualMemoryStat{}, err
	}
	return *v, nil
}

// CollectSwapMemory 采集交换区使用情况，返回 gopsutil 原始结构（值类型）。
func CollectSwapMemory(ctx context.Context) (mem.SwapMemoryStat, error) {
	v, err := mem.SwapMemoryWithContext(ctx)
	if err != nil || v == nil {
		return mem.SwapMemoryStat{}, err
	}
	return *v, nil
}

// CollectSwapDevices 采集交换区设备列表，返回 gopsutil 原始结构（值类型切片）。
func CollectSwapDevices(ctx context.Context) ([]mem.SwapDevice, error) {
	devices, err := mem.SwapDevicesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]mem.SwapDevice, 0, len(devices))
	for _, d := range devices {
		if d != nil {
			out = append(out, *d)
		}
	}
	return out, nil
}

// ================= Disk =================

// CollectDiskPartitions 采集磁盘分区列表；all=true 时包含伪文件系统在内的全部分区。
func CollectDiskPartitions(ctx context.Context, all bool) ([]disk.PartitionStat, error) {
	return disk.PartitionsWithContext(ctx, all)
}

// CollectDiskUsage 采集指定挂载点路径的空间用量，返回 gopsutil 原始结构（值类型）。
func CollectDiskUsage(ctx context.Context, path string) (disk.UsageStat, error) {
	v, err := disk.UsageWithContext(ctx, path)
	if err != nil || v == nil {
		return disk.UsageStat{}, err
	}
	return *v, nil
}

// CollectDiskIOCounters 采集磁盘 IO 计数器，key 为设备名；names 为空时返回全部设备。
func CollectDiskIOCounters(ctx context.Context, names ...string) (map[string]disk.IOCountersStat, error) {
	return disk.IOCountersWithContext(ctx, names...)
}

// ================= Host =================

// CollectHostInfo 采集主机与操作系统综合信息（含 hostname、os、platform、uptime 等），
// 返回 gopsutil 原始结构（值类型）。
func CollectHostInfo(ctx context.Context) (host.InfoStat, error) {
	v, err := host.InfoWithContext(ctx)
	if err != nil || v == nil {
		return host.InfoStat{}, err
	}
	return *v, nil
}

// CollectHostUsers 采集当前登录用户列表。
func CollectHostUsers(ctx context.Context) ([]host.UserStat, error) {
	return host.UsersWithContext(ctx)
}

// CollectVirtualization 采集虚拟化类型，直接返回 gopsutil 原始的 system、role 两个值。
func CollectVirtualization(ctx context.Context) (string, string, error) {
	return host.VirtualizationWithContext(ctx)
}

// CollectSensors 采集传感器温度信息（部分平台可能不支持，返回空列表）。
func CollectSensors(ctx context.Context) ([]host.TemperatureStat, error) {
	return host.SensorsTemperaturesWithContext(ctx)
}

// MachineUID 获取系统级唯一 ID（重启不变），取值优先级：
//   - Linux：/sys/class/dmi/id/product_uuid（硬件 UUID，需 root）→ /etc/machine-id（系统安装期稳定）
//   - macOS：IOPlatformUUID（硬件级）；Windows：注册表 MachineGuid（系统安装期稳定）
//
// 与 CollectHostInfo 里的 hostId 字段同源，但本函数会剔除 gopsutil 在 Linux 上的
// boot_id 兜底值（/proc/sys/kernel/random/boot_id 每次重启都会变），
// 保证返回值要么重启不变、要么报错，不会把不稳定值当稳定 ID 交出去。
// 注：VM 克隆镜像若未重新生成 machine-id/product_uuid，多台机器可能重复。
func MachineUID(ctx context.Context) (string, error) {
	raw, err := host.HostIDWithContext(ctx)
	if err != nil {
		return "", err
	}
	id := strings.ToLower(strings.TrimSpace(raw))
	if id == "" {
		return "", errors.New("无法获取系统唯一 ID")
	}

	// Linux 兜底值检查：两者相等说明只剩 boot_id，重启即变，不可用
	if runtime.GOOS == "linux" {
		if boot, rerr := os.ReadFile("/proc/sys/kernel/random/boot_id"); rerr == nil {
			if strings.ToLower(strings.TrimSpace(string(boot))) == id {
				return "", errors.New("仅能读到 boot_id 兜底值（重启会变），无法获得稳定唯一 ID")
			}
		}
	}

	return id, nil
}

// ================= Load =================

// CollectLoadAvg 采集系统平均负载（1/5/15 分钟），返回 gopsutil 原始结构（值类型）。
func CollectLoadAvg(ctx context.Context) (load.AvgStat, error) {
	v, err := load.AvgWithContext(ctx)
	if err != nil || v == nil {
		return load.AvgStat{}, err
	}
	return *v, nil
}

// CollectLoadMisc 采集其它负载信息（进程总数、上下文切换次数等），返回 gopsutil 原始结构（值类型）。
func CollectLoadMisc(ctx context.Context) (load.MiscStat, error) {
	v, err := load.MiscWithContext(ctx)
	if err != nil || v == nil {
		return load.MiscStat{}, err
	}
	return *v, nil
}

// ================= Net =================

// CollectNetIOCounters 采集网卡收发计数器；perNIC=true 按网卡拆分，false 返回总体。
func CollectNetIOCounters(ctx context.Context, perNIC bool) ([]net.IOCountersStat, error) {
	return net.IOCountersWithContext(ctx, perNIC)
}

// CollectNetInterfaces 采集网络接口列表。
func CollectNetInterfaces(ctx context.Context) (net.InterfaceStatList, error) {
	return net.InterfacesWithContext(ctx)
}

// CollectNetProtoCounters 采集指定协议（如 tcp/udp/ip/icmp）的计数器；
// 传入 nil 或空切片时使用默认协议列表。
func CollectNetProtoCounters(ctx context.Context, protocols []string) ([]net.ProtoCountersStat, error) {
	if len(protocols) == 0 {
		protocols = defaultProtoCounters
	}
	return net.ProtoCountersWithContext(ctx, protocols)
}

// CollectNetConnections 采集网络连接列表，kind 取值如 "tcp"/"udp"/"ip" 等（数据量较大）。
func CollectNetConnections(ctx context.Context, kind string) ([]net.ConnectionStat, error) {
	return net.ConnectionsWithContext(ctx, kind)
}

// ================= Process =================

// CollectPids 采集当前所有进程 PID 列表。
func CollectPids(ctx context.Context) ([]int32, error) {
	return process.PidsWithContext(ctx)
}

// CollectProcesses 采集全部进程对象列表。
// 注：process.Process 内含 sync.RWMutex，不可拷贝，故返回指针切片。
func CollectProcesses(ctx context.Context) ([]*process.Process, error) {
	return process.ProcessesWithContext(ctx)
}
