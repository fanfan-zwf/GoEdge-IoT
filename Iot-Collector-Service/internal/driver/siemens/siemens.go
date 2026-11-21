package siemens

import (
	"collector/internal/model"
	"collector/internal/repo/mysql"
	"collector/pkg/conv"
	"fmt"
	"log"
	"sync"
	"time"
)

type Config_type struct {
	Address                string        // 地址
	Retry_timeout          conv.Duration // 重试间隔（可选，默认3s）
	Connect_timeout        conv.Duration // 连接超时（可选，默认10s）
	Response_timeout       conv.Duration // 响应超时（可选，默认10s）
	Rack                   int           // 机架号
	Slot                   int           // 槽位号
	Delay_between_polls    conv.Duration // 轮询间隔（可选，默认0）
	ConnectionType         int           // 连接类型（可选，默认2） 1=PG编程设备 2=OP操作面板 3=Basic
	Max_reconnect_interval conv.Duration // 重连最大间隔（指数退避封顶，可选，默认30s）
	Local_TSAP             uint16        // 本地 TSAP 覆盖（snap7 原生类型 uint16，0=按 ConnectionType+Rack+Slot 自动推导）
	Remote_TSAP            uint16        // 远端 TSAP 覆盖（snap7 原生类型 uint16，如 CP 343-1 Advanced 需填其配置的 TSAP；0=自动推导）
}

// S7数据包点位结构体
type S7Points_type struct {
	Area          int // 存储区 0x81=I(输入) 0x82=Q(输出) 0x83=M(标志) 0x84=DB(数据块) 0x1C=C(计数器) 0x1D=T(定时器)（snap7 编码）
	DBNumber      int // DB号
	Start         int // 字节偏移 1byte = 8bit
	Child_Address int // 子地址（可选）
	// Type 采集类型：S7 PLC 侧原始数据类型，严格对齐 TIA Portal 变量声明类型；决定缓冲区长度与字节流解析方式
	// 代码值 ↔ Go 类型 ↔ TIA Portal 类型（宽度）：
	//   1  ↔ bool    ↔ Bool        （位，1 bit，需配合 Child_Address 指定位偏移）
	//   2  ↔ int8    ↔ SInt        （1 byte）
	//   3  ↔ uint8   ↔ USInt       （1 byte）
	//   4  ↔ int16   ↔ Int         （2 bytes，大端）
	//   5  ↔ uint16  ↔ UInt        （2 bytes，大端）
	//   6  ↔ int32   ↔ DInt        （4 bytes，大端 ABCD）
	//   7  ↔ uint32  ↔ UDInt       （4 bytes，大端 ABCD）
	//   8  ↔ int64   ↔ LInt        （8 bytes，大端 ABCDEFGH）
	//   9  ↔ uint64  ↔ ULInt       （8 bytes，大端 ABCDEFGH）
	//   10 ↔ int     ↔ —           （Go 原生别名，非 TIA 类型，按 LInt 宽度采集）
	//   11 ↔ uint    ↔ —           （Go 原生别名，非 TIA 类型，按 ULInt 宽度采集）
	//   12 ↔ float32 ↔ Real        （4 bytes，IEEE-754 大端 ABCD）
	//   13 ↔ float64 ↔ LReal       （8 bytes，IEEE-754 大端 ABCDEFGH）
	//   14 ↔ float   ↔ —           （Go 原生别名，非 TIA 类型，等价于 Real）
	//   15 ↔ string  ↔ String      （按 Value_Type 指定的缓冲区长度截取，到首个 0 字节结束）
	Type int
}

type Point_Config_type struct {
	Id  uint // 点位id
	Tag string

	RW_Cancel  int    // 读写方式读写方式 1：禁止； 2：只读； 3：只写； 4：读写；
	Value_Type int    // 输出类型：经过 Conversion 换算后，对外（数据库/上报/下游）呈现的最终类型 1：bool； 2：int8； 3：uint8； 4：int16； 5：uint16； 6：int32； 7：uint32； 8：int64； 9：uint64； 10：int； 11：uint； 12：float32； 13：float64； 14：float； 15：string；
	Conversion string // 换算表达式：连接【采集类型 Config.Type】与【输出类型 Value_Type】的桥梁，读取时正向换算（x 代表原始值），写入时反向换算；为空或 "0" 表示不做换算，仅做类型转换

	Config S7Points_type
}

type Siemens_S7 struct {
	Drive_Id uint
	Config   Config_type
	Points   []Point_Config_type

	// 以下派生字段与 Points 一起受 mu 保护：写入方（组包/点位更新/连接）在 Lock 下整体发布新引用，
	// 读取方（polling/Write）在 RLock 下取快照后基于本地引用操作，已发布对象不再被原地修改
	mu                           sync.RWMutex
	Read_Packet_S7DataItem       []S7DataItem
	Read_Packet_S7DataItem_Point map[int][]int // 读取数据包中点位的映射 map[Read_Packet_S7DataItem的index]Points的index

	client         *Snap7Client // snap7 客户端（受 lifeMu 保护：仅当前代循环发布/清空）
	closed         bool         // 连接关闭标志（受 lifeMu 保护）
	reconnectGen   uint64       // 重连/轮询循环代数：Connect/Close 递增，旧循环据此自我退出，杜绝并发重连
	retryCount     int          // 重连退避计数（仅重连循环使用）
	readFailStreak int          // 轮询连续读取失败轮数（达 s7ReconnectAfterFailures 才判定断线并重连）
	negotiatedPDU  int32        // 连接协商后的实际 PDU 长度
	lifeMu         sync.Mutex   // 保护 client / closed / reconnectGen 等连接生命周期状态

	Read_External_Mappings func([]model.Value_type) error // 外部映射回调
}

/*******************驱动初始化*******************/

// New 解析驱动配置和点位配置，构建读取数据包
func (c *Siemens_S7) New(Drive mysql.Drive_Config_Query_type, Points []mysql.Point_Config_Query_type) (err error) {

	c.Drive_Id = Drive.Id

	// 1. 解析驱动配置字符串
	c.Config, err = Drive_Config_Switch(Drive.Config)
	if err != nil {
		return fmt.Errorf("解析驱动配置失败: %w", err)
	}

	// 2. 解析点位配置
	var points []Point_Config_type
	for _, v := range Points {
		point, err := Point_Config_Switch(v.Config)
		if err != nil {
			log.Printf("WARN 点位【%s】 配置解析失败, 跳过: %v", v.Tag, err)
			continue
		}
		// 向后兼容：旧配置未提供采集类型时，回落到 Value_Type
		if point.Type == 0 {
			log.Printf("WARN 点位【%s】 未配置采集类型(Config.Type), 回落为输出类型 Value_Type【%d】, 建议在地址后缀补充 TIA 类型名", v.Tag, v.Value_Type)
			point.Type = v.Value_Type
		}
		points = append(points, Point_Config_type{
			Id:         v.Id,
			Tag:        v.Tag,
			RW_Cancel:  v.RW_Cancel,
			Value_Type: v.Value_Type,
			Conversion: v.Conversion,
			Config:     point,
		})
	}
	c.mu.Lock()
	c.Points = points
	c.mu.Unlock()

	// 3. 读取组包
	if err := c.Read_Packet(); err != nil {
		return fmt.Errorf("读取组包失败: %w", err)
	}

	return nil
}

// UpdatePoints 动态更新点位配置（重新解析点位+重组包，不断连接）
func (c *Siemens_S7) UpdatePoints(Points []mysql.Point_Config_Query_type) error {
	var points []Point_Config_type
	for _, v := range Points {
		point, err := Point_Config_Switch(v.Config)
		if err != nil {
			log.Printf("WARN 点位【%s】 UpdatePoints 配置解析失败, 跳过: %v", v.Tag, err)
			continue
		}
		// 向后兼容：旧配置未提供采集类型时，回落到 Value_Type
		if point.Type == 0 {
			log.Printf("WARN 点位【%s】 UpdatePoints 未配置采集类型(Config.Type), 回落为输出类型 Value_Type【%d】", v.Tag, v.Value_Type)
			point.Type = v.Value_Type
		}
		points = append(points, Point_Config_type{
			Id:         v.Id,
			Tag:        v.Tag,
			RW_Cancel:  v.RW_Cancel,
			Value_Type: v.Value_Type,
			Conversion: v.Conversion,
			Config:     point,
		})
	}
	c.mu.Lock()
	c.Points = points
	c.mu.Unlock()

	if err := c.Read_Packet(); err != nil {
		return fmt.Errorf("UpdatePoints 读取组包失败: %w", err)
	}

	c.mu.RLock()
	pointCnt, packCnt := len(c.Points), len(c.Read_Packet_S7DataItem)
	c.mu.RUnlock()
	log.Printf("INFO 驱动【%d】 点位动态更新完成, 点位数【%d】, 轮询包数【%d】", c.Drive_Id, pointCnt, packCnt)
	return nil
}

/*******************驱动连接*******************/

// packet 组包：筛选可读点位（RW_Cancel=2只读/4读写），按 Area+DBNumber 分组合并连续地址，
// 构建 Read_Packet_S7DataItem 和 Read_Packet_S7DataItem_Point
func (c *Siemens_S7) Read_Packet() error {
	// 读取共享配置快照（points/pdu），本地组包后整体发布，避免与轮询/写入并发访问共享字段
	c.mu.RLock()
	points := c.Points
	pdu := int(c.negotiatedPDU)
	c.mu.RUnlock()

	// 1. 筛选可读点位，转换为 S7PackPoint
	var packPoints []S7PackPoint
	for _, p := range points {
		if p.RW_Cancel != 2 && p.RW_Cancel != 4 {
			continue
		}
		byteLen := ValueTypeToByteLen(p.Config.Type) // 缓冲区长度按【采集类型】分配，避免输出类型与 PLC 实际宽度不一致时踩内存
		if byteLen <= 0 {
			log.Printf("WARN 点位【%s】 无效采集类型【%d】, 跳过", p.Tag, p.Config.Type)
			continue
		}
		packPoints = append(packPoints, S7PackPoint{
			Id:        p.Id,
			Area:      p.Config.Area,
			DBNumber:  p.Config.DBNumber,
			Start:     p.Config.Start,
			ByteLen:   byteLen,
			ValueType: p.Config.Type,
		})
	}

	if len(packPoints) == 0 {
		log.Printf("WARN 驱动【%d】 无有效的可读点位, 跳过组包", c.Drive_Id)
		return fmt.Errorf("无有效的可读点位")
	}

	// 2. 调用组包函数（优先使用协商后的实际 PDU；未连接时 negotiatedPDU 为 0，由组包内部回落默认 PDU）
	packs, err := PackS7AddressPackages(packPoints, pdu)
	if err != nil {
		log.Printf("WARN 驱动【%d】 S7 组包失败: %v", c.Drive_Id, err)
		return fmt.Errorf("S7 组包失败: %w", err)
	}

	// 3. 本地构建 S7DataItem 和点位映射（避免边写共享字段边被轮询读取）
	items := make([]S7DataItem, 0, len(packs))
	mapping := make(map[int][]int)

	for i, pack := range packs {
		wordLen := S7WLByte
		start := pack.Start // snap7 ReadMultiVars 直接使用字节偏移
		amount := pack.ByteLen
		dbNum := pack.DBNumber
		dataLen := pack.ByteLen
		if pack.Area == S7AreaTM || pack.Area == S7AreaCT {
			// Timer/Counter：Start 是编号，不乘 8；WordLen 用专用类型
			// Amount 是数量（不是字节数），DBNumber 必须为 0
			start = pack.Start
			amount = 1
			dbNum = 0
			dataLen = 2 // 每个 Timer/Counter 返回 2 字节
			if pack.Area == S7AreaTM {
				wordLen = S7WLTimer
			} else {
				wordLen = S7WLCounter
			}
		}
		item := S7DataItem{
			Area:     int32(pack.Area),
			WordLen:  int32(wordLen),
			DBNumber: int32(dbNum),
			Start:    int32(start),
			Amount:   int32(amount),
			Data:     make([]byte, dataLen),
		}
		items = append(items, item)

		// 构建 DataItem index → Points index 映射
		var pointIndices []int
		var pointIds []uint
		for _, id := range pack.Ids {
			for j, p := range points {
				if p.Id == id {
					pointIndices = append(pointIndices, j)
					pointIds = append(pointIds, p.Id)
					break
				}
			}
		}
		mapping[i] = pointIndices

		// 诊断日志：点位 ID、DB、起始/结束字节地址、报文参数
		endByte := pack.Start + pack.ByteLen - 1
		log.Printf("DEBUG 驱动【%d】 DataItem【%d】 点位IDs=%v Area=0x%X DB=%d Start=%d End=%d Amount=%d WordLen=0x%X BufLen=%d",
			c.Drive_Id, i, pointIds, item.Area, item.DBNumber, item.Start, endByte, item.Amount, item.WordLen, len(item.Data))
	}

	// 4. 整体发布：一次性替换引用，旧的切片/映射不再被写入，正使用快照的轮询/写入协程不受影响
	c.mu.Lock()
	c.Read_Packet_S7DataItem = items
	c.Read_Packet_S7DataItem_Point = mapping
	c.mu.Unlock()

	return nil
}

/*******************驱动连接*******************/

// loopAlive 判断某代(重连/轮询)循环是否仍应继续：未被 Close、且仍是最新一代。
// 旧代循环在任意检查点拿到 false 即自我退出，从根本上杜绝「多条重连循环并发刷连接」。
func (c *Siemens_S7) loopAlive(gen uint64) bool {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	return !c.closed && gen == c.reconnectGen
}

// snapshotClient 在 lifeMu 下取当前 snap7 客户端句柄，可能为 nil（未连接或已关闭）。
func (c *Siemens_S7) snapshotClient() *Snap7Client {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	return c.client
}

// Connect 绑定外部映射回调，建立连接，启动轮询
func (c *Siemens_S7) Connect(Read_External_Mappings func([]model.Value_type) error) error {
	c.Read_External_Mappings = Read_External_Mappings

	// 递增连接代数并作废旧循环：任何仍在运行的旧重连/轮询循环都会在下一检查点自我退出，
	// 避免保存配置(ResetConfig→Connect)时新旧循环并发刷同一 PLC 造成连接风暴。
	c.lifeMu.Lock()
	c.closed = false
	c.reconnectGen++
	gen := c.reconnectGen
	c.lifeMu.Unlock()

	err := c.connect(gen)
	if err != nil {
		// 首次连接失败，后台持续重连直到成功或被关闭
		log.Printf("WARN 驱动【%d】 首次连接失败: %v, 进入重连循环", c.Drive_Id, err)
		go func() {
			if reconnectErr := c.reconnect(gen); reconnectErr != nil {
				log.Printf("ERROR 驱动【%d】 重连终止: %v", c.Drive_Id, reconnectErr)
				c.Error_External_Mappings(reconnectErr.Error())
				return
			}
			c.polling(gen)
		}()
		return nil
	}
	go c.polling(gen)
	return nil
}

// buildAutoTSAP 依据连接类型 + 机架 + 槽位自动推导本地/远端 TSAP（snap7 约定）：
//
//	local  = ConnectionType << 8
//	remote = ConnectionType << 8 + Rack*0x20 + Slot
//
// 常见取值（OP 连接 connType=2）：S7-300(Rack0/Slot2)→0x0202；S7-1200/1500(Rack0/Slot1)→0x0201；S7-400(Rack0/Slot3)→0x0203
// 传入：[connType] 1=PG/2=OP/3=Basic, [rack] 机架号(0~7), [slot] 槽位号(0~31)
// 返回：local/remote TSAP；参数非法时返回 error
func buildAutoTSAP(connType, rack, slot int) (local, remote uint16, err error) {
	switch connType {
	case ConnTypePG, ConnTypeOP, ConnTypeBasic:
	default:
		return 0, 0, fmt.Errorf("无效连接类型【%d】(需 1=PG/2=OP/3=Basic)", connType)
	}
	if rack < 0 || rack > 7 {
		return 0, 0, fmt.Errorf("机架号超出范围【%d】(0~7)", rack)
	}
	if slot < 0 || slot > 31 {
		return 0, 0, fmt.Errorf("槽位号超出范围【%d】(0~31)", slot)
	}
	base := uint16(connType) << 8
	offset := uint16(rack)*0x20 + uint16(slot)
	return base, base + offset, nil
}

// connect 建立 S7 TCP 连接（使用 snap7 C 动态库）；gen 为发起方循环代数，用于防止旧代循环发布失效连接
// TSAP 与 PDU 均采用「自动计算 / 协商」：
//   - TSAP：默认由 ConnectionType + Rack + Slot 自动推导（见 buildAutoTSAP）；若配置了 Local_TSAP/Remote_TSAP 则以其为准（手动覆盖，用于 CP 343-1 Advanced 等需匹配已配置连接 TSAP 的设备）；
//   - PDU：连接前以固定请求上限 s7RequestPDU 作为「请求值」，连接后取 PLC 实际协商值重组包，协商失败则回落请求值。
func (c *Siemens_S7) connect(gen uint64) error {
	client := NewSnap7Client()

	// 连接类型（1=PG 2=OP 3=Basic），默认 OP
	connType := c.Config.ConnectionType
	if connType <= 0 {
		connType = ConnTypeOP
	}

	// --- TSAP：默认由 ConnectionType + Rack + Slot 自动推导；若配置了 Local_TSAP/Remote_TSAP 则手动覆盖
	// （用于 CP 343-1 Advanced 等「S7 连接需预配置、使用自定义 TSAP」的设备；留空则回落自动推导，兼容 Lean/集成网口） ---
	autoLocal, autoRemote, err := buildAutoTSAP(connType, c.Config.Rack, c.Config.Slot)
	if err != nil {
		client.Destroy()
		return err
	}
	localTSAP, remoteTSAP := autoLocal, autoRemote
	if c.Config.Local_TSAP != 0 {
		localTSAP = c.Config.Local_TSAP
	}
	if c.Config.Remote_TSAP != 0 {
		remoteTSAP = c.Config.Remote_TSAP
	}
	if localTSAP != autoLocal || remoteTSAP != autoRemote {
		log.Printf("INFO 驱动【%d】 使用手动 TSAP local=0x%04X remote=0x%04X（自动推导为 0x%04X/0x%04X）", c.Drive_Id, localTSAP, remoteTSAP, autoLocal, autoRemote)
	}
	if err := client.SetConnectionParams(c.Config.Address, localTSAP, remoteTSAP); err != nil {
		client.Destroy()
		return fmt.Errorf("设置 TSAP 失败 local【0x%04X】remote【0x%04X】: %w", localTSAP, remoteTSAP, err)
	}

	// --- PDU：连接前声明请求上限，PLC 会下调到自身支持值（协商结果即最终 PDU） ---
	if err := client.SetParam(ParamPDURequest, s7RequestPDU); err != nil {
		log.Printf("WARN 驱动【%d】 设置请求 PDU【%d】失败: %v", c.Drive_Id, s7RequestPDU, err)
	}

	// 超时参数（毫秒）
	if c.Config.Connect_timeout > 0 {
		if err := client.SetParam(ParamSendTimeout, int32(time.Duration(c.Config.Connect_timeout).Milliseconds())); err != nil {
			log.Printf("WARN 驱动【%d】 设置发送超时失败: %v", c.Drive_Id, err)
		}
	}
	if c.Config.Response_timeout > 0 {
		if err := client.SetParam(ParamRecvTimeout, int32(time.Duration(c.Config.Response_timeout).Milliseconds())); err != nil {
			log.Printf("WARN 驱动【%d】 设置接收超时失败: %v", c.Drive_Id, err)
		}
	}

	log.Printf("INFO siemens_s7 连接参数: %+v", c.Config)
	if err := client.Connect(); err != nil {
		// 连接失败时 TCP 可能已半开（三次握手完成但 ISO/COTP 协商未完成）：
		// 先 Disconnect() 主动发 FIN，尽快让对端回收这条会话，再 Destroy() 释放本地资源，
		// 避免频繁重连在本端/对端堆积半开连接。
		_ = client.Disconnect()
		client.Destroy()
		return fmt.Errorf("siemens_s7 连接失败: %w", err)
	}

	// 发布连接：仅当代循环仍有效时才占用 c.client，否则丢弃这条连接交还新一代
	c.lifeMu.Lock()
	if c.closed || gen != c.reconnectGen {
		c.lifeMu.Unlock()
		_ = client.Disconnect()
		client.Destroy()
		return fmt.Errorf("连接结果已作废（被新一代连接取代或驱动已关闭）")
	}
	c.client = client
	c.lifeMu.Unlock()

	// --- PDU：读取协商后的实际值，<=0 时回落请求上限 ---
	_, pduLen := client.GetPduLength()
	if pduLen <= 0 {
		pduLen = s7RequestPDU
	}
	c.mu.Lock()
	c.negotiatedPDU = pduLen
	c.mu.Unlock()

	// 用协商后的实际 PDU 重组包（替换 New() 中基于配置值的初始组包）
	if err := c.Read_Packet(); err != nil {
		log.Printf("WARN 驱动【%d】 协商PDU【%d】重组包失败: %v", c.Drive_Id, pduLen, err)
	}

	log.Printf("INFO 驱动【%d】 连接成功, 地址【%s】, rack【%d】, slot【%d】, 请求PDU【%d】, 协商PDU【%d】", c.Drive_Id, c.Config.Address, c.Config.Rack, c.Config.Slot, s7RequestPDU, pduLen)
	return nil
}

// s7DefaultMaxReconnectInterval 重连指数退避最大间隔的兜底默认值（配置项 Max_reconnect_interval 缺省时使用）。
const s7DefaultMaxReconnectInterval = 30 * time.Second

// 弱网自适应参数：在丢包/高抖动链路上尽量保住已建立的 S7 会话。
// 核心思想——重连要在烂链路上重新赌一次 ISO/COTP 握手，代价远高于一次读超时，
// 所以读失败先「原地重试」，连续多轮仍失败才判定真断线；重连时每轮退避前先做快速探测，
// 以便链路瞬时好转的瞬间立刻连上，而不是干等一轮退避。
const (
	s7ReadRetries            = 1                      // 单批读失败时在同一连接上的原地重试次数（不含首次）
	s7ReadRetryPause         = 500 * time.Millisecond // 原地重试前的短暂停顿
	s7ReconnectAfterFailures = 2                      // 连续多少个轮询周期重试后仍失败，才判定连接已断并触发重连
	s7ConnectBurst           = 2                      // 每轮重连内、退避等待前连续快速建连尝试的次数
	s7ConnectBurstPause      = 1 * time.Second        // 快速重连尝试之间的间隔
)

// reconnect 掉线重连：以 Retry_timeout 为初始间隔做指数退避，每次失败间隔翻倍，
// 封顶配置项 Max_reconnect_interval；gen 为本循环代数，一旦被 Close 或新一代取代即自我退出。
func (c *Siemens_S7) reconnect(gen uint64) error {
	interval := time.Duration(c.Config.Retry_timeout)
	if interval <= 0 {
		interval = 3 * time.Second
	}
	maxInterval := time.Duration(c.Config.Max_reconnect_interval)
	if maxInterval <= 0 {
		maxInterval = s7DefaultMaxReconnectInterval
	}

	for {
		if !c.loopAlive(gen) {
			return fmt.Errorf("驱动已关闭或重连循环已被新一代取代")
		}

		// 关闭并释放旧连接（在 lifeMu 下摘走 c.client，避免与其它协程并发操作同一句柄）
		c.lifeMu.Lock()
		old := c.client
		c.client = nil
		c.lifeMu.Unlock()
		if old != nil {
			old.Disconnect()
			old.Destroy()
		}

		// 每轮退避等待前，做有限次「快速建连尝试」：弱网链路上每次握手成败带有偶然性，
		// 连续快速重试能在链路瞬时好转的瞬间立刻抓住窗口连上，而不必干等一轮指数退避。
		var err error
		for attempt := 1; attempt <= s7ConnectBurst; attempt++ {
			if !c.loopAlive(gen) {
				return fmt.Errorf("驱动已关闭或重连循环已被新一代取代")
			}
			log.Printf("INFO 驱动【%d】 尝试重连(%d/%d), 地址【%s】 ...", c.Drive_Id, attempt, s7ConnectBurst, c.Config.Address)
			if err = c.connect(gen); err == nil {
				log.Printf("INFO 驱动【%d】 重连成功", c.Drive_Id)
				c.retryCount = 0 // 成功后复位退避
				return nil
			}
			// 建连失败：连接结果可能因被新一代取代而作废，此时立即退出不再重试
			if !c.loopAlive(gen) {
				return err
			}
			if attempt < s7ConnectBurst {
				c.Error_External_Mappings(err.Error())
				time.Sleep(s7ConnectBurstPause)
			}
		}

		// 本轮所有快速尝试均失败 → 上报错误状态并做指数退避等待
		c.Error_External_Mappings(err.Error())
		log.Printf("WARN 驱动【%d】 重连失败: %v, %s 后重试", c.Drive_Id, err, interval)
		time.Sleep(interval)
		c.retryCount++ // 重试计数

		// 指数退避：失败间隔翻倍，封顶 maxInterval（配置项 Max_reconnect_interval）
		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}

// Close 关闭驱动连接
func (c *Siemens_S7) Close() error {
	c.lifeMu.Lock()
	c.closed = true
	c.reconnectGen++ // 作废当前重连/轮询循环，令其在下一检查点自我退出
	cli := c.client
	c.client = nil
	c.lifeMu.Unlock()

	if cli != nil {
		cli.Disconnect()
		cli.Destroy()
	}
	c.Error_External_Mappings("驱动连接已关闭")
	return nil
}

// readBatch 一次 ReadMultiVars 调用的 item 索引区间 [start, end)
type readBatch struct {
	start int
	end   int
}

// computeReadBatches 依据协商 PDU 将 Read_Packet_S7DataItem 分批，保证每次 ReadMultiVars 均合法：
//   - 报文不超 PDU（PDU 是整包含 S7 头）：Σ(item 数据 + 协议开销) + 基础头 ≤ 协商 PDU；
//   - 每批 item 数 ≤ s7MaxItemsPerRead(20，snap7 MaxVars 硬限制，超过 C 层直接报错)；
//   - 单个 item 独占一批（组包阶段已限制其数据区 ≤ PDU−基础头−开销）。
func (c *Siemens_S7) computeReadBatches(items []S7DataItem, maxPDU int) []readBatch {
	if maxPDU <= 0 {
		maxPDU = s7DefaultPDUSize
	}
	// 先扣除固定基础头，得到「数据 + 每 item 开销」可用的净空间
	usable := maxPDU - s7PduBaseOverhead
	if usable <= 0 {
		usable = maxPDU // 极小 PDU 兑底（理论不发生）
	}

	batches := make([]readBatch, 0, len(items)/s7MaxItemsPerRead+1)
	batchStart := 0
	batchBytes := 0

	for i, item := range items {
		itemTotal := int(item.Amount) + s7ItemOverhead
		// 加入当前 item 会超出净空间或超过 MaxVars → 先封存当前批（至少保留 1 个 item）
		if i > batchStart && (batchBytes+itemTotal > usable || i-batchStart >= s7MaxItemsPerRead) {
			batches = append(batches, readBatch{start: batchStart, end: i})
			batchStart = i
			batchBytes = 0
		}
		batchBytes += itemTotal
	}
	if batchStart < len(items) {
		batches = append(batches, readBatch{start: batchStart, end: len(items)})
	}
	return batches
}

// batchS7Items 将任意 S7DataItem 切片按「协商 PDU 净空间」+「MaxVars(20)」双重限制分批，
// 供写入路径（WriteMultiVars / bool 预读 ReadMultiVars）使用，避免单批报文超 PDU 或 item 数超限。
func (c *Siemens_S7) batchS7Items(items []S7DataItem, maxPDU int) [][]S7DataItem {
	if maxPDU <= 0 {
		maxPDU = s7DefaultPDUSize
	}
	usable := maxPDU - s7PduBaseOverhead
	if usable <= 0 {
		usable = maxPDU
	}

	batches := make([][]S7DataItem, 0, len(items)/s7MaxItemsPerRead+1)
	cur := make([]S7DataItem, 0, s7MaxItemsPerRead)
	curBytes := 0
	for _, it := range items {
		total := int(it.Amount) + s7ItemOverhead
		if len(cur) > 0 && (curBytes+total > usable || len(cur) >= s7MaxItemsPerRead) {
			batches = append(batches, cur)
			cur = make([]S7DataItem, 0, s7MaxItemsPerRead)
			curBytes = 0
		}
		cur = append(cur, it)
		curBytes += total
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// polling 轮询读取所有数据包，掉线自动重连；gen 为本循环代数，一旦被 Close 或新一代取代即退出
// 按协商 PDU 分批发送 ReadMultiVars，每批同时受「数据+开销≤PDU净空间」与「item 数≤20」约束
func (c *Siemens_S7) polling(gen uint64) {
	c.mu.RLock()
	initLen := len(c.Read_Packet_S7DataItem)
	initPDU := int(c.negotiatedPDU)
	c.mu.RUnlock()
	if initLen == 0 {
		log.Printf("WARN 驱动【%d】 无有效轮询包, 退出轮询", c.Drive_Id)
		return
	}

	log.Printf("INFO 驱动【%d】 轮询启动, 包数【%d】, PDU【%d】", c.Drive_Id, initLen, initPDU)

	for {
		if !c.loopAlive(gen) {
			log.Printf("INFO 驱动【%d】 已关闭或被新连接取代, 退出轮询", c.Drive_Id)
			return
		}

		// 取本轮一致性快照：一次读取 items/mapping/points/pdu，整轮基于同一份引用；
		// 即便期间点位热更新重组并发布新数组，也只影响下一轮，绝不出现陈旧索引越界
		c.mu.RLock()
		items := c.Read_Packet_S7DataItem
		mapping := c.Read_Packet_S7DataItem_Point
		points := c.Points
		pdu := int(c.negotiatedPDU)
		c.mu.RUnlock()

		if len(items) == 0 {
			time.Sleep(time.Duration(c.Config.Delay_between_polls))
			continue
		}

		batches := c.computeReadBatches(items, pdu)

		// 逐批读取：单批失败先在同一连接上原地重试，只有连续多轮仍失败才判定连接已断并触发重连。
		// 弱网下单次读超时极其常见，若一超时就拆连接，等于每次都在烂链路上重新赌握手，反而更不稳定。
		readErr := false
		var lastErr error
		for _, b := range batches {
			// 兜底：快照内恒成立，防御性避免任何越界
			if b.start < 0 || b.start >= b.end || b.end > len(items) {
				log.Printf("WARN 驱动【%d】 分批[%d,%d)超出快照长度(%d)，跳过", c.Drive_Id, b.start, b.end, len(items))
				continue
			}
			batchItems := items[b.start:b.end]
			ok := false
			for attempt := 0; attempt <= s7ReadRetries; attempt++ {
				cli := c.snapshotClient()
				if cli == nil {
					lastErr = fmt.Errorf("连接未就绪/已关闭")
					break // 句柄已没了，原地重试无意义，交给上层判定重连
				}
				if code := cli.ReadMultiVars(batchItems); code != nil {
					lastErr = code
					if attempt < s7ReadRetries {
						time.Sleep(s7ReadRetryPause)
						continue
					}
					break
				}
				// 读取成功：回写本批结果到快照（供本轮解析；底层缓冲与已发布数组同源，下一轮取新快照）
				for j, item := range batchItems {
					items[b.start+j] = item
				}
				ok = true
				break
			}
			if !ok {
				readErr = true
				break
			}
		}

		if readErr {
			c.readFailStreak++
			log.Printf("WARN 驱动【%d】 读取失败(已原地重试, 连续第 %d/%d 轮): %v", c.Drive_Id, c.readFailStreak, s7ReconnectAfterFailures, lastErr)
			c.Error_External_Mappings(lastErr.Error())
			// 未达判定阈值：保留当前连接，短暂等待后进入下一轮再试，避免误拆尚可用的会话
			if c.readFailStreak < s7ReconnectAfterFailures && c.loopAlive(gen) {
				time.Sleep(time.Duration(c.Config.Delay_between_polls))
				continue
			}
			// 连续多轮失败 → 判定连接已断，触发重连
			c.readFailStreak = 0
			if reconnectErr := c.reconnect(gen); reconnectErr != nil {
				log.Printf("ERROR 驱动【%d】 重连最终失败, err【%v】, 退出轮询", c.Drive_Id, reconnectErr)
				c.Error_External_Mappings(reconnectErr.Error())
				return
			}
			// 重连成功后回到循环顶，重新取快照并重算分批（PDU 可能变化）
			continue
		}
		c.readFailStreak = 0 // 本轮全部批次成功，复位连续失败计数

		// 逐个 DataItem 检查结果（全部基于本轮快照）
		for j := 0; j < len(items); j++ {
			item := &items[j]
			if item.Result != 0 {
				errMsg := CliErrorText(item.Result)
				c.Error_External_Mappings_byPacketIndex(j, errMsg.Error())
				continue
			}

			readList := c.analysis(j, item.Data, items, mapping, points)
			if len(readList) > 0 && c.Read_External_Mappings != nil {
				if cbErr := c.Read_External_Mappings(readList); cbErr != nil {
					log.Printf("WARN 驱动【%d】 数据回调失败, err【%v】", c.Drive_Id, cbErr)
				}
			}
		}

		time.Sleep(time.Duration(c.Config.Delay_between_polls))
	}
}

// analysis 解析单个 DataItem 的读取数据，返回点位值列表
func (c *Siemens_S7) analysis(packetIndex int, data []byte, items []S7DataItem, mapping map[int][]int, points []Point_Config_type) []model.Value_type {
	pointIndices, ok := mapping[packetIndex]
	if !ok {
		return nil
	}

	if packetIndex < 0 || packetIndex >= len(items) {
		return nil
	}
	item := items[packetIndex]
	var readList []model.Value_type
	now := time.Now()

	for _, pIdx := range pointIndices {
		if pIdx < 0 || pIdx >= len(points) {
			continue
		}
		point := points[pIdx]

		read := model.Value_type{
			PointId: point.Id,
			Tag:     point.Tag, // 点位标识：下游（InfluxDB 历史数据）按此建序列
			Type:    point.Value_Type,
			Msg:     "ok",
			Time:    now,
		}

		byteOffset := point.Config.Start - int(item.Start) // snap7 Start 直接是字节偏移
		byteLen := ValueTypeToByteLen(point.Config.Type)   // 读取宽度按【采集类型】
		if byteLen <= 0 {
			read.Msg = fmt.Sprintf("点位【%s】 采集类型无效, 类型编码【%d】", point.Tag, point.Config.Type)
			log.Printf("WARN %s, 跳过", read.Msg)
			readList = append(readList, read)
			continue
		}
		if byteOffset < 0 || byteOffset+byteLen > len(data) {
			read.Msg = fmt.Sprintf("点位【%s】 字节偏移越界, offset【%d】, byteLen【%d】, dataLen【%d】", point.Tag, byteOffset, byteLen, len(data))
			log.Printf("WARN %s, 跳过", read.Msg)
			readList = append(readList, read)
			continue
		}

		// 1. 按【采集类型】从字节流解析原始值（S7 固定大端序）
		var v any
		switch point.Config.Type {
		case conv.TYPE_BOOL: // bool → 提取 data[byteOffset] 的第 Child_Address 位
			v = conv.Get_list_index(
				conv.BytesToBool(conv.Get_list_index(data, byteOffset, 1)),
				point.Config.Child_Address, 1)[0]
		case conv.TYPE_INT8: // int8
			v = int8(conv.Get_list_index(data, byteOffset, 1)[0])
		case conv.TYPE_UINT8: // uint8
			v = conv.Get_list_index(data, byteOffset, 1)[0]
		case conv.TYPE_INT16: // int16 大端 BA
			v = conv.Get_list_index(
				conv.BytesToInt16(conv.Get_list_index(data, byteOffset, 2), conv.BA),
				0, 1)[0]
		case conv.TYPE_UINT16: // uint16 大端 BA
			v = conv.Get_list_index(
				conv.BytesToUint16(conv.Get_list_index(data, byteOffset, 2), conv.BA),
				0, 1)[0]
		case conv.TYPE_INT32: // int32 大端 ABCD
			v = conv.Get_list_index(
				conv.BytesToInt32(conv.Get_list_index(data, byteOffset, 4), conv.ABCD),
				0, 1)[0]
		case conv.TYPE_UINT32: // uint32 大端 ABCD
			v = conv.Get_list_index(
				conv.BytesToUint32(conv.Get_list_index(data, byteOffset, 4), conv.ABCD),
				0, 1)[0]
		case conv.TYPE_INT64: // int64 大端 ABCDEFGH
			v = conv.Get_list_index(
				conv.BytesToInt64(conv.Get_list_index(data, byteOffset, 8), conv.ABCDEFGH),
				0, 1)[0]
		case conv.TYPE_UINT64: // uint64 大端 ABCDEFGH
			v = conv.Get_list_index(
				conv.BytesToUint64(conv.Get_list_index(data, byteOffset, 8), conv.ABCDEFGH),
				0, 1)[0]
		case conv.TYPE_INT: // int → 按 int64
			v = int(conv.Get_list_index(
				conv.BytesToInt64(conv.Get_list_index(data, byteOffset, 8), conv.ABCDEFGH),
				0, 1)[0])
		case conv.TYPE_UINT: // uint → 按 uint64
			v = uint(conv.Get_list_index(
				conv.BytesToUint64(conv.Get_list_index(data, byteOffset, 8), conv.ABCDEFGH),
				0, 1)[0])
		case conv.TYPE_FLOAT32: // float32 大端 ABCD
			v = conv.Get_list_index(
				conv.BytesToFloat32(conv.Get_list_index(data, byteOffset, 4), conv.ABCD),
				0, 1)[0]
		case conv.TYPE_FLOAT64: // float64 大端 ABCDEFGH
			v = conv.Get_list_index(
				conv.BytesToFloat64(conv.Get_list_index(data, byteOffset, 8), conv.ABCDEFGH),
				0, 1)[0]
		case conv.TYPE_FLOAT: // float → 按 float32
			v = conv.Get_list_index(
				conv.BytesToFloat32(conv.Get_list_index(data, byteOffset, 4), conv.ABCD),
				0, 1)[0]
		case conv.TYPE_STRING: // string → 截取到第一个 0 字节
			raw := conv.Get_list_index(data, byteOffset, byteLen)
			end := len(raw)
			for i, b := range raw {
				if b == 0 {
					end = i
					break
				}
			}
			v = string(raw[:end])
		default:
			read.Msg = fmt.Sprintf("点位【%s】 不支持的采集类型, 类型编码【%d】", point.Tag, point.Config.Type)
		}

		if v != nil {
			// 2. 换算 + 类型转换：采集类型 → Conversion → 输出类型
			result, err := conv.ConvertWithCalc(v, point.Config.Type, point.Value_Type, point.Conversion)
			if err != nil {
				read.Msg = fmt.Sprintf("点位【%s】 %s, 采集类型【%d】→输出类型【%d】", point.Tag, err.Error(), point.Config.Type, point.Value_Type)
			} else {
				read.Value = result
			}
		}
		readList = append(readList, read)
	}

	return readList
}

/*******************错误上报*******************/

// Error_External_Mappings 上报所有点位的错误状态
func (c *Siemens_S7) Error_External_Mappings(msg string) error {
	if c.Read_External_Mappings == nil {
		return nil
	}
	c.mu.RLock()
	points := c.Points
	c.mu.RUnlock()
	readList := make([]model.Value_type, 0, len(points))
	for _, point := range points {
		readList = append(readList, model.Value_type{
			PointId: point.Id,
			Tag:     point.Tag,
			Type:    point.Value_Type,
			Msg:     msg,
			Time:    time.Now(),
		})
	}
	return c.Read_External_Mappings(readList)
}

// Error_External_Mappings_byPacketIndex 按数据包索引上报对应点位的错误状态
func (c *Siemens_S7) Error_External_Mappings_byPacketIndex(packetIndex int, msg string) error {
	if c.Read_External_Mappings == nil {
		return nil
	}
	c.mu.RLock()
	pointIndices, ok := c.Read_Packet_S7DataItem_Point[packetIndex]
	points := c.Points
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	readList := make([]model.Value_type, 0, len(pointIndices))
	for _, pIdx := range pointIndices {
		if pIdx < 0 || pIdx >= len(points) {
			continue
		}
		point := points[pIdx]
		readList = append(readList, model.Value_type{
			PointId: point.Id,
			Tag:     point.Tag,
			Type:    point.Value_Type,
			Msg:     msg,
			Time:    time.Now(),
		})
	}
	return c.Read_External_Mappings(readList)
}

/*******************写入*******************/

// Write 写入点位值：校验可写性 → 按 Area+DBNumber 动态组包 → AGReadMulti 读取当前值（bool 位操作需要）→ 更新缓冲区 → AGWriteMulti 批量写入
// 功能：传入待写入点位值，动态构建写入包，bool 类型先读后写（位操作不破坏同字节其他位）
// 传入：values — 待写入的点位值切片
// 返回：error — 写入过程中的错误
func (c *Siemens_S7) Write(values []model.Value_type) error {
	// 1. 校验每个写入值对应的点位是否存在且允许写入
	tagPointsMap := make(map[uint]model.Value_type)
	for _, v := range values {
		tagPointsMap[v.PointId] = v
	}
	// 取共享配置快照：写入过程中即使点位热更新重组，也不影响本次写入的一致性
	c.mu.RLock()
	points := c.Points
	pdu := int(c.negotiatedPDU)
	c.mu.RUnlock()

	// 取当前连接句柄快照：Close/重连可能将 c.client 置 nil，统一取一次并判空，避免 nil 解引用
	cli := c.snapshotClient()
	if cli == nil {
		return fmt.Errorf("驱动未连接或已关闭, 写入取消")
	}
	for _, p := range points {
		if _, exists := tagPointsMap[p.Id]; !exists {
			continue
		}
		if p.RW_Cancel != 3 && p.RW_Cancel != 4 {
			return fmt.Errorf("点位【%s】 不允许写入, RW_Cancel【%d】（需为 3=只写 或 4=读写）", p.Tag, p.RW_Cancel)
		}
	}

	// 2. 按 Area+DBNumber 分组，动态构建写入 DataItem
	type writeGroup struct {
		item         S7DataItem
		pointIndices []int
		hasBool      bool // 本包是否包含 bool 点位（需要先读取）
	}
	var groups []writeGroup

	type groupKey struct{ Area, DBNumber int }
	writePoints := make(map[groupKey][]int) // key → Points index 列表
	for i, p := range points {
		if p.RW_Cancel != 3 && p.RW_Cancel != 4 {
			continue
		}
		if _, exists := tagPointsMap[p.Id]; !exists {
			continue
		}
		key := groupKey{p.Config.Area, p.Config.DBNumber}
		writePoints[key] = append(writePoints[key], i)
	}

	for key, indices := range writePoints {
		// 构建组包输入点位
		var packPoints []S7PackPoint
		for _, idx := range indices {
			p := points[idx]
			byteLen := ValueTypeToByteLen(p.Config.Type) // 写缓冲按【采集类型】分配
			if byteLen <= 0 {
				continue
			}
			packPoints = append(packPoints, S7PackPoint{
				Id: p.Id, Area: p.Config.Area, DBNumber: p.Config.DBNumber,
				Start: p.Config.Start, ByteLen: byteLen,
			})
		}
		if len(packPoints) == 0 {
			continue
		}

		// 调用组包（合并连续地址）
		// 优先使用协商后的实际 PDU；未连接时 negotiatedPDU 为 0，由组包内部回落默认 PDU
		packs, err := PackS7AddressPackages(packPoints, pdu)
		if err != nil {
			return fmt.Errorf("S7 写入组包失败: %w", err)
		}

		for _, pack := range packs {
			area := key.Area
			wordLen := S7WLByte
			start := pack.Start // snap7 ReadMultiVars 直接使用字节偏移
			amount := pack.ByteLen
			dbNum := key.DBNumber
			dataLen := pack.ByteLen
			if area == S7AreaTM || area == S7AreaCT {
				start = pack.Start // Timer/Counter：Start 是编号
				amount = 1
				dbNum = 0
				dataLen = 2
				if area == S7AreaTM {
					wordLen = S7WLTimer
				} else {
					wordLen = S7WLCounter
				}
			}
			item := S7DataItem{
				Area: int32(area), WordLen: int32(wordLen), DBNumber: int32(dbNum),
				Start: int32(start), Amount: int32(amount),
				Data: make([]byte, dataLen),
			}
			// 构建本包涉及的点位索引
			var pointIndices []int
			for _, id := range pack.Ids {
				for j, p := range points {
					if p.Id == id {
						if _, exists := tagPointsMap[p.Id]; exists {
							pointIndices = append(pointIndices, j)
						}
						break
					}
				}
			}
			// 检查本包是否包含 bool 点位（以【采集类型】判定，因为位操作依赖 PLC 侧原始布局）
			hasBool := false
			for _, idx := range pointIndices {
				if points[idx].Config.Type == conv.TYPE_BOOL {
					hasBool = true
					break
				}
			}
			groups = append(groups, writeGroup{item: item, pointIndices: pointIndices, hasBool: hasBool})
		}
	}

	if len(groups) == 0 {
		return nil
	}

	// 3. 提取需要先预读的 bool 包（非 bool 包无需预读，第 6 步直接从 groups 取全部写入）
	var boolItems []S7DataItem
	var boolIndices []int // boolItems 在 groups 中的索引

	for i, g := range groups {
		if g.hasBool {
			boolItems = append(boolItems, g.item)
			boolIndices = append(boolIndices, i)
		}
	}

	// 4. bool 包：ReadMultiVars 先读取当前值（位操作需要保留同字节其他位）；按 PDU + MaxVars 分批
	if len(boolItems) > 0 {
		processed := make([]S7DataItem, 0, len(boolItems)) // 顺序累积各批读取结果，用于回写
		for bi, batch := range c.batchS7Items(boolItems, pdu) {
			if err := cli.ReadMultiVars(batch); err != nil {
				return fmt.Errorf("ReadMultiVars 批量读取失败 batch=%d: %w", bi, err)
			}
			processed = append(processed, batch...)
		}
		// 回写读取结果到 groups（processed 与 boolItems 同序，故索引对齐 boolIndices）
		for i, gIdx := range boolIndices {
			groups[gIdx].item = processed[i]
		}
	}

	// 5. 逐包更新缓冲区
	for i := range groups {
		if err := writeUpdateBuffer(&groups[i].item, groups[i].pointIndices, points, tagPointsMap); err != nil {
			log.Printf("ERROR 写入准备失败, Area【%d】, DB【%d】, Start【%d】, err【%v】", groups[i].item.Area, groups[i].item.DBNumber, groups[i].item.Start, err)
		}
	}

	// 6. WriteMultiVars 批量写入 PLC（bool 包已在前一步预读合并，此处按 PDU + MaxVars 分批统一写入）
	allItems := make([]S7DataItem, 0, len(groups))
	for _, g := range groups {
		allItems = append(allItems, g.item)
	}
	for bi, batch := range c.batchS7Items(allItems, pdu) {
		if err := cli.WriteMultiVars(batch); err != nil {
			return fmt.Errorf("WriteMultiVars 批量写入失败 batch=%d: %w", bi, err)
		}
	}
	return nil
}

// writeUpdateBuffer 更新单个 DataItem 的缓冲区（数据已由 AGReadMulti 读取完毕）
// 功能：逐个点位校验时间、类型，将转换后的值写入缓冲区；bool 类型用位操作
// 传入：item — 待更新的 S7DataItem 指针
//
//	pointIndices — 本包涉及的点位在 points 中的索引
//	points — 全部点位配置
//	tagPointsMap — 待写入值映射（pointId → value）
//
// 返回：error — 校验或转换失败时返回错误
func writeUpdateBuffer(item *S7DataItem, pointIndices []int, points []Point_Config_type, tagPointsMap map[uint]model.Value_type) error {
	now := time.Now()

	for _, pIdx := range pointIndices {
		if pIdx < 0 || pIdx >= len(points) {
			continue
		}
		point := points[pIdx]

		v, exists := tagPointsMap[point.Id]
		if !exists {
			continue
		}

		// 时间校验
		if !v.Time.IsZero() {
			duration := now.Sub(v.Time)
			if duration > 5*time.Second {
				return fmt.Errorf("点位【%s】 写入值时间间隔过长, 间隔【%s】", point.Tag, duration)
			}
		}

		// 类型校验
		if v.Type != point.Value_Type {
			return fmt.Errorf("点位【%s】 配置类型与写入值类型不匹配, 配置类型【%d】, 值类型【%d】", point.Tag, point.Value_Type, v.Type)
		}

		byteOffset := point.Config.Start - int(item.Start) // snap7 Start 直接是字节偏移
		byteLen := ValueTypeToByteLen(point.Config.Type)   // 写入宽度按【采集类型】
		if byteOffset < 0 || byteLen <= 0 || byteOffset+byteLen > len(item.Data) {
			return fmt.Errorf("点位【%s】 字节偏移越界, offset【%d】, byteLen【%d】, bufLen【%d】", point.Tag, byteOffset, byteLen, len(item.Data))
		}

		// 反向换算 + 类型转换：输出类型 → Conversion⁻¹ → 采集类型
		convertedValue, err := conv.ConvertReverse(v.Value, point.Value_Type, point.Config.Type, point.Conversion)
		if err != nil {
			return fmt.Errorf("点位【%s】 写入值类型转换失败, 输出类型【%d】→采集类型【%d】, err【%w】", point.Tag, point.Value_Type, point.Config.Type, err)
		}

		if point.Config.Type == conv.TYPE_BOOL {
			// bool 类型：位操作（保留同字节其他位）
			boolVal, ok := convertedValue.(bool)
			if !ok {
				return fmt.Errorf("点位【%s】 值类型断言失败, 期望【bool】, 实际【%T】", point.Tag, convertedValue)
			}
			if point.Config.Child_Address > 7 {
				return fmt.Errorf("点位【%s】 子地址超出范围, child_address【%d】, 最大【7】", point.Tag, point.Config.Child_Address)
			}
			conv.SetBit(item.Data, byteOffset*8+point.Config.Child_Address, boolVal)
		} else {
			// 非 bool 类型：按【采集类型】将值编为字节流（S7 大端序）并写入缓冲区
			switch point.Config.Type {
			case conv.TYPE_INT8:
				val, ok := convertedValue.(int8)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【int8】, 实际【%T】", point.Tag, convertedValue)
				}
				item.Data[byteOffset] = byte(val)
			case conv.TYPE_UINT8:
				val, ok := convertedValue.(uint8)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【uint8】, 实际【%T】", point.Tag, convertedValue)
				}
				item.Data[byteOffset] = val
			case conv.TYPE_INT16:
				val, ok := convertedValue.(int16)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【int16】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Int16ToBytes([]int16{val}, conv.BA))
			case conv.TYPE_UINT16:
				val, ok := convertedValue.(uint16)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【uint16】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Uint16ToBytes([]uint16{val}, conv.BA))
			case conv.TYPE_INT32:
				val, ok := convertedValue.(int32)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【int32】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Int32ToBytes([]int32{val}, conv.ABCD))
			case conv.TYPE_UINT32:
				val, ok := convertedValue.(uint32)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【uint32】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Uint32ToBytes([]uint32{val}, conv.ABCD))
			case conv.TYPE_INT64:
				val, ok := convertedValue.(int64)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【int64】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Int64ToBytes([]int64{val}, conv.ABCDEFGH))
			case conv.TYPE_UINT64:
				val, ok := convertedValue.(uint64)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【uint64】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Uint64ToBytes([]uint64{val}, conv.ABCDEFGH))
			case conv.TYPE_INT:
				val, ok := convertedValue.(int)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【int】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Int64ToBytes([]int64{int64(val)}, conv.ABCDEFGH))
			case conv.TYPE_UINT:
				val, ok := convertedValue.(uint)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【uint】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Uint64ToBytes([]uint64{uint64(val)}, conv.ABCDEFGH))
			case conv.TYPE_FLOAT32:
				val, ok := convertedValue.(float32)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【float32】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Float32ToBytes([]float32{val}, conv.ABCD))
			case conv.TYPE_FLOAT64:
				val, ok := convertedValue.(float64)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【float64】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Float64ToBytes([]float64{val}, conv.ABCDEFGH))
			case conv.TYPE_FLOAT:
				val, ok := convertedValue.(float64)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【float64】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], conv.Float32ToBytes([]float32{float32(val)}, conv.ABCD))
			case conv.TYPE_STRING:
				val, ok := convertedValue.(string)
				if !ok {
					return fmt.Errorf("点位【%s】 值类型断言失败, 期望【string】, 实际【%T】", point.Tag, convertedValue)
				}
				copy(item.Data[byteOffset:], []byte(val))
			default:
				return fmt.Errorf("点位【%s】 不支持的采集类型（写入）【%d】", point.Tag, point.Config.Type)
			}
		}
	}

	return nil
}
