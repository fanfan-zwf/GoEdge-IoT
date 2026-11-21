# `pkg/sysinfo` — 系统信息采集：gopsutil 的逐项薄封装

上级文档：[服务总览 README](../../README.md)

本包把 `github.com/shirou/gopsutil/v3` 的 **cpu / mem / disk / host / load / net / process** 七个子模块逐项包成 24 个 `CollectXxx` 函数，设计原则写在文件头（`sysinfo.go:4-7`）：**gopsutil 有多少采集能力就提供多少个独立函数，直接返回原始结果、不拼接、不聚合**。这样对外的 `/system/info` 接口能按 `type` 参数逐一路由，MQTT 通路也能按需单独采集其中几项。

消费方只有两处：[internal/handler/system.go](../../internal/handler/README.md)（HTTP，全 24 项几乎都用上）与 [internal/service/internalmqtt/publish.go](../../internal/service/internalmqtt/README.md)（MQTT 遥测主题表，只挑了 13 项）。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `sysinfo.go` | 227 | 全部 24 个导出采集函数 + `DefaultCPUInterval` 常量 + `defaultProtoCounters` 默认协议表 + `MachineUID` |

包内只依赖标准库与 gopsutil，**不 import 项目任何其它包**，因此可以被任意层安全引用（包括 `repo` 与 `driver`）。

---

## 二、⚠️ 最容易踩的坑

1. **每个函数都要传 `context.Context`，且超时由调用方负责。**
   本包不设超时。gopsutil 在 Windows / macOS 上某些调用（尤其 `CollectSensors`、`CollectNetConnections`、`CollectProcesses`）可能明显阻塞。HTTP 侧 `handler/system.go` 用的是 `ctx.Request.Context()`，MQTT 侧自行 `context.WithTimeout`。

2. **`CollectCPUPercent` 会阻塞 `interval` 时长。**
   `DefaultCPUInterval = 1s`（`sysinfo.go:29`），handler 与 MQTT 都用它 → **`/system/info?type=cpu/percent` 单次至少 1 秒**。把它挂在高频轮询上会直接拖垮前端体验。传 `interval <= 0` 则不阻塞、返回"自启动以来的均值"，但那不是当前负载。

3. **返回值刻意用「值类型」而不是指针。**
   `CollectVirtualMemory` / `CollectSwapMemory` / `CollectDiskUsage` / `CollectHostInfo` / `CollectLoadAvg` / `CollectLoadMisc` 都做了 `if err != nil || v == nil { return T{}, err }` 的解引用保护（如 `sysinfo.go:60-66`）。
   **副作用：`err == nil` 且结构体全零是可能出现的**——gopsutil 返回 `(nil, nil)` 时本包也会返回 `(零值, nil)`。判定成功与否不能靠"字段是否为 0"。

4. **`CollectProcesses` 返回 `[]*process.Process`（指针），不能改成值切片。**
   `sysinfo.go:224` 的注释说明了原因：`process.Process` 内含 `sync.RWMutex`，拷贝会被 `go vet` 判为复制锁，且行为未定义。同时**这个函数会把全机进程对象一次性构造出来**，在容器宿主或高负载机器上开销很大，别放进常规轮询。

5. **`CollectProcesses` / `CollectNetConnections` 的返回结构不可直接 JSON 序列化。**
   指针 + 内含锁 + 循环引用风险，直接塞进 `Response_Use` 的 `Data` 会失败或输出巨大。消费方 `handler/system.go` 里 `type=process` 是单独分支处理的，扩展时注意。

6. **`CollectDiskUsage(path)` 的 `path` 必须是已挂载路径，且会原样透传给 `statfs`。**
   handler 侧 `path := ctx.Query("path")`（`system.go:165`）—— 空串/不存在路径会返回 error（表现为接口 500/520），而不是"整盘统计"。MQTT 侧硬编码 `"/"`。这是本包**唯一接受任意用户输入**的函数，属于潜在的路径探测面，RBAC 上要把 `/system/info?type=disk/usage` 收紧。

7. **`CollectNetConnections(kind)` 的 `kind` 不做白名单校验。**
   `net.ConnectionsWithContext` 期望 `"tcp" / "udp" / "ip" / "icmp" / "all"` 等，非法值返回 error（`system.go:191` 处由 handler 给默认值）。数据量大，`"all"` 在有大量短连接的生产机上很重。

8. **`CollectNetProtoCounters(nil)` 用内置默认协议表，而不是"全部协议"。**
   `sysinfo.go:32`：`{"tcp", "udp", "udp6", "ip", "icmp", "icmp6"}`。**注意这个列表没有 `tcp6`** —— 只看 `tcp` 会漏掉 IPv6 监听端口的计数。要显式传 `protocols` 参数（handler 支持 `?protocols=tcp,tcp6`，逗号分隔）。

9. **`MachineUID` 目前无任何调用方（零引用）。**
   `sysinfo.go:140-168`。它存在的意义是给"设备指纹 / 授权绑定"预留：调 `host.HostIDWithContext` 后**主动剔除 gopsutil 在 Linux 上的 `boot_id` 兜底值**（把结果与 `/proc/sys/kernel/random/boot_id` 比对，相等就报错而不是返回一个重启即变的值）。将来做机器绑定时应复用它，不要直接用 `CollectHostInfo().HostID`——那个字段在容器里就是 boot_id。

10. **VM 克隆后 `MachineUID` 可能重复。**
    函数头注释（`sysinfo.go:147`）已声明：克隆镜像若未重新生成 `machine-id` / `product_uuid`，多台机器 ID 相同。做唯一性约束前要在部署侧确认。另外 `/sys/class/dmi/id/product_uuid` **需要 root** 才能读，非 root 进程会自动退到 `/etc/machine-id`。

11. **`CollectSensors` 在很多平台返回空列表而非 error。**
    虚拟机、容器、多数云主机都没有温度传感器。前端要把"空数组"当正常态处理，不要显示成故障。

12. **`defaultProtoCounters` 与 `DefaultCPUInterval` 一个是包级 `var`、一个是 `const`。**
    `defaultProtoCounters` 是**可变切片**且未加锁，理论上能被同包代码改写；当前只读，扩展时不要在函数里修改它。

---

## 三、函数总览（按 gopsutil 子模块）

所有函数第一参数均为 `ctx context.Context`。

### CPU（`sysinfo.go:34-55`）

| 函数 | 返回 | 对应 gopsutil | 备注 |
| --- | --- | --- | --- |
| `CollectCPUInfo` | `[]cpu.InfoStat` | `cpu.InfoWithContext` | 静态信息（型号、频率、核编号） |
| `CollectCPUCounts(logical bool)` | `int` | `cpu.CountsWithContext` | `true` 逻辑核 / `false` 物理核 |
| `CollectCPUPercent(interval, perCPU bool)` | `[]float64` | `cpu.PercentWithContext` | **阻塞 interval**；`perCPU` 决定切片长度 |
| `CollectCPUTimes(perCPU bool)` | `[]cpu.TimesStat` | `cpu.TimesWithContext` | user/system/idle 等**累计值**，需自己两次相减才是差值 |

### Mem（`57-90`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectVirtualMemory` | `mem.VirtualMemoryStat`（值） | 物理内存 |
| `CollectSwapMemory` | `mem.SwapMemoryStat`（值） | 交换区总量/用量 |
| `CollectSwapDevices` | `[]mem.SwapDevice` | 逐个设备；内部已跳过 `nil` 元素 |

### Disk（`92-111`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectDiskPartitions(all bool)` | `[]disk.PartitionStat` | `all=true` 含 proc/sysfs 等伪文件系统，条目会非常多 |
| `CollectDiskUsage(path string)` | `disk.UsageStat`（值） | 唯一带用户输入参数的函数，见坑 6 |
| `CollectDiskIOCounters(names ...string)` | `map[string]disk.IOCountersStat` | key 是设备名；不传 names 返回全部 |

### Host（`113-168`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectHostInfo` | `host.InfoStat`（值） | hostname / os / platform / uptime / **HostID** 等综合信息，前端概览页主力 |
| `CollectHostUsers` | `[]host.UserStat` | 当前登录用户（含 tty、idle 时长） |
| `CollectVirtualization` | `(system, role string, err)` | **唯一一个返回两个业务值的函数**，如 `("kvm", "guest")` |
| `CollectSensors` | `[]host.TemperatureStat` | 常为空，见坑 11 |
| `MachineUID` | `(string, error)` | 重启不变的机器 ID；已剔除 boot_id；当前零调用（坑 9） |

### Load（`170-188`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectLoadAvg` | `load.AvgStat`（值） | 1/5/15 分钟负载；**Windows 上不支持** |
| `CollectLoadMisc` | `load.MiscStat`（值） | 进程总数、上下文切换等 |

### Net（`190-214`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectNetIOCounters(perNIC bool)` | `[]net.IOCountersStat` | 收发字节/包计数（累计值） |
| `CollectNetInterfaces` | `net.InterfaceStatList` | 网卡名、MAC、地址 |
| `CollectNetProtoCounters(protocols []string)` | `[]net.ProtoCountersStat` | 传空用默认 6 协议（坑 8） |
| `CollectNetConnections(kind string)` | `[]net.ConnectionStat` | 数据量大，谨慎 |

### Process（`216-227`）

| 函数 | 返回 | 备注 |
| --- | --- | --- |
| `CollectPids` | `[]int32` | 便宜，只有 PID |
| `CollectProcesses` | `[]*process.Process` | 指针切片，不可拷贝，重开销（坑 4、5） |

---

## 四、对外暴露形态

本包自己不注册路由。两项消费方式：

1. **HTTP**：`GET /system/info?type=<分支>`，`handler/system.go:132-201` 的 `switch` 覆盖 `host`、`host/users`、`host/virtualization`、`sensors`、`cpu`、`cpu/cores`、`cpu/percent`、`cpu/times`、`mem`、`mem/swap`、`mem/swapdevices`、`disk`、`disk/usage`、`disk/io`、`load`、`load/misc`、`net`、`net/interfaces`、`net/protocols`、`net/connections`、`process`、`process/pids` 等 type；布尔参数统一走 `queryBool(ctx, name, default)`，识别 `"1"/"true"/"yes"` 与 `"0"/"false"/"no"`（`system.go:109-111`）。
2. **MQTT**：`internalmqtt/publish.go` 里一张 `{"主题后缀", func(ctx) (any, error)}` 表，取 `host`、`cpu`、`cpu/cores`、`disk`、`disk/io`、`disk/usage`、`load`、`mem`、`mem/swap`、`net`、`net/interfaces`、`process/pids`、`sensors`、`cpu/percent` 等子集（比 HTTP 少，且不暴露 `net/connections`、`process` 这类重接口）。

> 两边的 type 集合**不一致**：新增采集项时要么两处都加，要么在文档里说明差异，否则会出现"HTTP 能查、MQTT 查不到"的困惑。

权限：`/system/info/#` 是 RBAC 主题里唯一使用了 `#` 通配的资源（见 [pkg/rbac README](../rbac/README.md)），意味着"系统信息"整块可以一次性授权或一次性封禁，但**做不到只封 `type=disk/usage` 这一项**。

---

## 五、排障速查

| 现象 | 定位 | 处理 |
| --- | --- | --- |
| `/system/info?type=cpu/percent` 慢 | 设计如此，采样 1s | 见坑 2；前端降低轮询频率 |
| CPU 使用率一直是 0 | gopsutil 在容器里读宿主 `/proc/stat` 的行为 | 确认是否需要在 cgroup 层取值；`type=cpu/times` 自己算差值 |
| 内存/负载返回全零但没有报错 | gopsutil 返回 `(nil, nil)`（坑 3） | 检查平台支持；`CollectXxx` 不做二次判断 |
| `type=disk/usage` 报 500 | `path` 查询参数为空或不存在 | 传绝对路径，如 `?path=/` |
| `type=net/protocols` 少了 tcp6 | 默认协议表未含 tcp6 | `?protocols=tcp,tcp6,udp,udp6` |
| `type=sensors` 空数组 | 无传感器（VM/容器） | 正常，见坑 11 |
| `type=load` 在 Windows 报错 | gopsutil 平台限制 | 本服务主要面向 Linux 边缘设备 |
| 进程列表接口 OOM / 极慢 | `CollectProcesses` 全量构造 | 改用 `CollectPids`；或做采样与缓存 |
| 机器 ID 每次重启都变 | 拿到了 boot_id | 用 `MachineUID` 而不是 `CollectHostInfo().HostID`（坑 9） |

---

## 六、扩展指引

**新增一个采集项（例如 `host/SysInfo`）**
1. 在本文件按分节加一个函数，**保持"一个函数对应一个 gopsutil 调用、返回原始结果"**的风格，不要在此聚合多个来源；
2. 若 gopsutil 返回指针，套用 `if err != nil || v == nil { return T{}, err }` 的解引用保护模板；
3. 到 `handler/system.go` 的 `switch` 加一个 `case`；需要布尔/字符串参数时用 `queryBool` / `ctx.Query`；
4. 视需要同步 `internalmqtt/publish.go` 的主题表；
5. 权限：确认新的 `type` 落在 `/system/info/#` 覆盖范围内（当前是），若拆成独立路径则要补 `t_permission` 种子数据。

**性能原则**：本包保持"无缓存、无状态、无后台协程"。若要给前端做高频轮询优化，缓存请加在 handler 层（带 TTL），**不要**下沉到 pkg —— 一旦包内出现状态，就无法再被其它服务安全复用。

---

## 七、相关文档

- [服务总览 README](../../README.md)
- [internal/handler](../../internal/handler/README.md) — `GET /system/info` 的 type 路由与参数解析
- [internal/service/internalmqtt](../../internal/service/internalmqtt/README.md) — MQTT 遥测主题表与采集节流
- [pkg/rbac](../rbac/README.md) — `/system/info/#` 这条唯一使用 `#` 的权限主题
