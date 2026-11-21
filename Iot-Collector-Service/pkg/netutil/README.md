# `pkg/netutil` — 通用 TCP/UDP 客户端（带发送队列与惰性重连）

上级文档：[服务总览 README](../../README.md)

一个把 TCP 与 UDP 统一成同一个 `Client` 接口的连接工具：异步发送队列 + 惰性重连 + 读超时 + 残留数据清理。

> ⚠️ **本包在当前代码库中零引用**（`grep -rn netutil --include=*.go` 除自身外无命中）。协议驱动的联网由 `goburrow/modbus` 与 snap7 CGO 各自负责（见 [internal/driver/modbus](../../internal/driver/modbus/README.md)、[internal/driver/siemens](../../internal/driver/siemens/README.md)）。因此本包更接近"预置工具 / 试验田"，下方坑清单既是使用须知，也是接入前的整改清单。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `tcp_or_udp.go` | 485 | 哨兵错误、`ConnType`/`Config`/`Client` 定义、`baseClient` 公共实现、`TcpClient` / `UdpClient` 各自的 dial/Send/Receive/Close、末尾一个示例 `main()` |

---

## 二、⚠️ 最容易踩的坑

1. **`NewClient` 吞掉了首次连接的错误。**
   `tcp_or_udp.go:89`：`_ = cli.Reconnect(nil)` 后直接返回 `Client`。所以拿到的对象**可能是未连上的**，`err` 无处可查。要判断连通性只能靠第一次 `Send`/`Receive` 的返回。

2. **`Send` 不复制字节，直接把调用方的 slice 塞进队列。**
   `:240` / `:369`：`case t.sendQueue <- data:`。真正写网络是在 `sendWorker` 协程里。**如果调用方复用编码缓冲区（工业协议里非常常见的写法），队列里排的所有报文会同时变成最后一帧的内容。** 接入方每次 `Send` 必须自己传一个新建的切片。

3. **TCP 没有分帧/组包能力，`Receive` 只 `Read` 一次。**
   `:301`：`n, err := conn.Read(buf)` 后直接 `return buf[:n]`。TCP 是字节流，一次 `Read` 可能只拿到半个响应（也可能粘了两个）。工业设备一问一答场景需要**自己按长度或结束符组包**，本包不提供。

4. **重连是"懒触发"的，只有 `sendWorker` 会重连。**
   `:253-272` 仅在"队列里有数据要发且 `conn == nil`"时才尝试 `Reconnect`。**没有后台保活/探测**：只读不写的连接断开后永远不会自己恢复，`Receive` 也不会触发重连（只会返回 `ErrNotConnected`）。

5. **重连失败会丢弃当前这条报文。**
   `:263-271`：`Reconnect` 失败 → 打印 → `time.Sleep(ReconnectInterval)` → `continue`。`continue` 意味着**这条数据直接丢了**，不会重试。队列满时同理（`Send` 返回 `ErrQueueFull`，由调用方决定重试）。

6. **`Send` 与 `Close` 有竞态，可能 panic。**
   `Send` 先 `isClose()` 再 `sendQueue <- data`；`Close` 里 `close(t.sendQueue)`（`:330`）。两个协程交错时会撞上 **send on closed channel**。当前无调用方所以未暴露，接入时要么加互斥、要么改成"Close 不 close channel，靠 `closed` 标志让 worker 退出"。

7. **`OnConnected` 的回调一辈子只可能排一次。**
   `startCallback` 用 `b.callbackOnce`（`:56`、`:139`）保护 `time.AfterFunc`，而 `sync.Once` **永不复位** —— 断线重连成功后不会再触发。且它用的是 `AfterFunc` 单次定时器，不是注释里说的"定时循环执行"（`:44` 的接口注释与实现不符）。

8. **`startCallback` 读 `callbackFunc` 不加锁，`OnConnected` 写它加锁 → 数据竞争。**
   `:136` 的 `if b.callbackFunc == nil` 在 `b.mu` 之外；`reconnectInternal` 调用 `startCallback` 时也没有持锁。并发下 `-race` 会报。

9. **UDP 的 `udpConn` 字段完全不受锁保护。**
   `dial()` 写 `u.udpConn`（`:359`）不加锁，`sendWorker`（`:381`、`:387`）与 `Receive`（`:398`）读它也不加锁；而 `baseClient.conn` 走的是 `setConn/getConn` 的读写锁 —— **同一份连接被两套互不同步的状态记录**。

10. **`Reconnect` 对 UDP 会留下悬空的 `udpConn`。**
    `reconnectInternal` 只清 `b.conn`（`:119-124`），**不会**把 `u.udpConn` 置 nil。于是 `sendWorker` 的 `if u.udpConn == nil` 判断为假 → 往**已 Close 的旧 conn** 上写。UDP 换地址重连后行为异常即源于此。

11. **`UdpClient.Close` 与 `TcpClient.Close` 收尾字段不一致。**
    UDP 版关 `udpConn` 并置 nil，但**没有关/清 `baseClient.conn`**（`:429-432`）；TCP 版关 `t.conn`（`:323-326`）。`ClearRecvBuffer`、`getConn` 走的都是 `baseClient.conn`，UDP 上会拿到旧值。

12. **日志用 `fmt.Println` + emoji，不走项目的 `log`。**
    `:129`、`:262`、`:265`、`:277`、`:383`、`:389`。这些输出**不会进 `configs/config.yaml` 里 `LOG.Path` 配置的日志文件**，而是直写 stdout。接入到服务里会造成"日志文件里查不到任何连接痕迹"。（项目现有各包统一用 `log.Printf`，见根 README 的日志约定。）

13. **`cfg.Type` 无校验：非 `TCP` 的一切取值都当 UDP。**
    `:79`：`if cfg.Type == TCP {...} else {UDP}`。写成 `"Tcp"`、`""`、`"tcp4"` 都会静默变成 UDP 客户端。

14. **`Config.SendQueueSize` 为 0 时队列无缓冲。**
    `make(chan []byte, 0)` 是同步通道，`Send` 的 `select ... default` 会在 worker 没有正好阻塞在收的地方时立刻返回 `ErrQueueFull`。必须显式给正数（示例用 50）。

15. **`reconnectInternal` 的 `closed` 检查是 TOCTOU。**
    `:104-109` 取锁检查后立刻放锁，再去关旧连接、拨号。与 `Close()` 并发时可能在已关闭的客户端上重建出连接（泄漏）。

16. **`ErrWriteFailed` 是死哨兵。**
    `:18` 定义，全包从未返回过（写失败时返回的是底层 `net` 错误，见 `:276-281`、`:388-390`）。上层若 `errors.Is(err, ErrWriteFailed)` 判断写失败，永远不会命中。

17. **`main()` 在 `package netutil` 里不会被执行**（`:441`）。
    它只是一段"可编译的用法示例"。与 [configsync](../../internal/service/configsync/README.md) 里那个同名函数一样属于遗留物，正式整理时应移进 `_test.go` 或 doc comment。

---

## 三、类型与 API

### 3.1 哨兵错误（`:11-19`）

| 错误 | 文本 | 何时返回 |
| --- | --- | --- |
| `ErrNotConnected` | 未建立网络连接 | `Receive` 时 conn 为 nil |
| `ErrInvalidLength` | 接收长度必须大于0 | `Receive(maxLen<=0, …)` |
| `ErrReceiveTimeout` | 接收数据超时 | 读超时（`net.Error.Timeout()`） |
| `ErrQueueFull` | 发送队列已满 | `Send` 时队列满（非阻塞入队） |
| `ErrClosed` | 客户端已关闭 | `Close` 之后的 `Send`/`Reconnect`/`dial` |
| `ErrConnectTimeout` | 建立连接超时 | 拨号超时（TCP 由 `net.Dialer.Timeout` 控制；UDP `DialUDP` 一般不返回超时） |
| ~~`ErrWriteFailed`~~ | 写入数据失败，连接已断开 | **从未返回**，见坑 16 |

### 3.2 配置

```go
type ConnType string
const ( TCP ConnType = "tcp"; UDP ConnType = "udp" )   // 其它取值会被当成 UDP（坑 13）

type Config struct {
    Type              ConnType      // TCP / UDP
    RemoteAddr        string        // "ip:port"
    LocalAddr         string        // 可空；UDP 侧解析失败被忽略（见下）
    Reconnect         bool          // 仅控制 sendWorker 里断线后是否重拨
    ReconnectInterval time.Duration // 重连失败后的 sleep；为 0 表示不 sleep
    ConnectTimeout    time.Duration // 传给 net.Dialer.Timeout（仅 TCP 生效）
    SendQueueSize     int           // 发送队列容量，0 = 无缓冲（坑 14）
}
```

UDP 分支里 `lAddr, _ := net.ResolveUDPAddr("udp", cfg.LocalAddr)`（`:344`）**丢弃了错误**：`LocalAddr` 写错不会报错，而是静默按"任意本地地址"绑定。

### 3.3 接口

```go
type Client interface {
    Send(data []byte) error                                      // 非阻塞入队，不复制数据
    Receive(maxLen int, timeout time.Duration) ([]byte, error)   // 单次 Read；timeout<=0 表示不设限
    ClearSendQueue()                                             // 丢弃队列里所有待发数据
    ClearRecvBuffer(discardSize int, waitTimeout time.Duration)  // 排空连接上的残留脏数据
    Close() error                                                // 幂等
    OnConnected(at time.Duration, callback func())               // 连上后 at 时长触发一次（只一次）
    Reconnect(newCfg *Config) error                              // newCfg==nil 表示用原配置重连
}
```

`baseClient` 提供公共实现，`TcpClient` / `UdpClient` 嵌入它并各自覆写 `dial/Send/Receive/Close/Reconnect`。注意 `ClearSendQueue`、`ClearRecvBuffer`、`OnConnected`、`startCallback`、`isClose`、`getConn`、`setConn`、`reconnectInternal` 都是 `baseClient` 上的共用方法。

### 3.4 三个易被误用的共用方法

| 方法 | 行为细节 |
| --- | --- |
| `ClearSendQueue` | `select` + `default` 循环抽干队列；**不通知 worker**，worker 可能已经取走一条在写 |
| `ClearRecvBuffer` | 建 `discardSize` 缓冲，设 `waitTimeout` 读截止，循环 `Read` 到出错才 break，最后复位截止时间为无限。**`waitTimeout<=0` 会让 deadline 立即到期，几乎一条都清不掉**；且 TCP 上这会真实消耗数据，若与业务 `Receive` 并发会互相抢包 |
| `Reconnect(nil)` | 保留原 cfg；`Reconnect(&newCfg)` 整体替换 cfg（**包括 `SendQueueSize`——但队列 channel 已按旧容量创建，不会重建**） |

---

## 四、生命周期

```
NewClient(cfg)
  ├─ 建 baseClient + make(chan []byte, cfg.SendQueueSize)
  ├─ go {Tcp,Udp}Client.sendWorker()      ← 常驻协程，靠 range channel 退出
  └─ _ = Reconnect(nil)                    ← 错误被丢弃（坑 1）
        └─ reconnectInternal
             closed? → ErrClosed
             newCfg != nil → cfgMu 保护下整体替换 cfg
             关旧 conn、置 b.conn=nil
             dial()
             成功 → fmt.Println("✅ 连接/重连成功") + startCallback()   ← Once，只排一次（坑 7）

sendWorker（后台）
  for raw := range sendQueue
     isClose → return
     conn==nil:
        !cfg.Reconnect → continue（丢弃该条）
        Reconnect 失败 → sleep(ReconnectInterval) → continue（丢弃该条，坑 5）
     conn.Write(raw)
        失败 → Close conn、conn=nil、continue（该条已丢弃）

Close
  持 b.mu；closed=true（幂等）
  关 conn（TCP）/ udpConn（UDP，坑 11）
  callbackTimer.Stop()
  close(sendQueue) → worker 抽干剩余后 return
```

---

## 五、接入前的整改清单

按优先级（当前零引用，所以是"整改完再上"而不是"边用边绕"）：

1. 把 `fmt.Println/Printf` 全量换成 `log.Printf`（坑 12）；
2. `Send` 内部复制一份 `data`，或者在接口文档里加粗"调用方必须传独占切片"（坑 2）；
3. `Close` 改成不 `close(sendQueue)`，改用 `done` channel + `closed` 标志让 worker 退出，消掉 send-on-closed panic（坑 6）；
4. 统一 UDP 的连接状态：去掉 `udpConn` 字段，全部走 `baseClient.conn` + `getConn/setConn`（坑 9、10、11）；
5. `callbackOnce` 改为可复位的定时器管理（`Reconnect` 成功时 `Stop` 旧 timer 再排新的），让 `OnConnected` 名副其实（坑 7）；
6. `startCallback` 内的 `callbackFunc` 读取纳入 `b.mu`（坑 8）；
7. `NewClient` 改为 `NewClient(cfg) (Client, error)`，至少把首次拨号错误回传（坑 1）；
8. 补 `cfg.Type` 白名单校验与 `SendQueueSize <= 0` 的默认值（坑 13、14）；
9. 删掉 `ErrWriteFailed`，或者真的在写失败路径上包装它（坑 16）；
10. 删除 `main()`，示例改写为 `Example` 测试（坑 17）；
11. 若用于一问一答协议，补 `Request(req, maxResp, timeout)` 这类"发送+等待响应"的组合方法，并明确 TCP 组包由调用方负责（坑 3）。

---

## 六、排障速查

| 现象 | 定位 |
| --- | --- |
| 明明没连上，`NewClient` 却没报错 | 坑 1，错误被 `_ =` 丢弃 |
| 发出去的报文内容变成最后一条 | 坑 2，缓冲区被复用 |
| 响应时多时少、半包 | 坑 3，无分帧 |
| 设备重启后一直连不上，直到下一次 `Send` | 坑 4，重连是懒触发 |
| 偶发 `panic: send on closed channel` | 坑 6 |
| `OnConnected` 断线重连后不再触发 | 坑 7，`sync.Once` 不复位 |
| UDP 换地址重连后发不出数据 | 坑 10，`udpConn` 悬空 |
| 连接/重连痕迹不在日志文件里 | 坑 12，`fmt.Println` 写的是 stdout |
| 频繁 `发送队列已满` | `SendQueueSize` 太小，或 worker 卡在 `ReconnectInterval` 的 sleep 里 |
| `ClearRecvBuffer` 之后还是读到旧数据 | `waitTimeout` 传了 0 或过小（坑 3.4） |
| `-race` 报在 `callbackFunc` / `udpConn` | 坑 8、9 |

---

## 七、相关文档

- [服务总览 README](../../README.md)
- [internal/driver/modbus](../../internal/driver/modbus/README.md) — 实际在用的 TCP 类协议联网方式（`goburrow/modbus` 自管连接）
- [internal/driver/siemens](../../internal/driver/siemens/README.md) — 实际在用的 S7 联网方式（snap7 CGO）
- [internal/infra/mqtt](../../internal/infra/mqtt/README.md) — 项目里另一条长连接通路及其重连策略（可对比参考）
- [internal/config](../../internal/config/README.md) — `LOG.Path` 等日志落盘约定（与坑 12 直接相关）
