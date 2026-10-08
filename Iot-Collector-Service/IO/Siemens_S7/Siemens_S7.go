package siemenss7

import (
	"fmt"
	"log"
	"main/IO/conv"
	"main/IO/manager/fullConfig"
	"main/db/mysql"
	"time"
)

type Config_type struct {
	Address             string        // 地址
	Retry_timeout       time.Duration // 重试间隔（可选，默认3s）
	Connect_timeout     time.Duration // 连接超时（可选，默认10s）
	Response_timeout    time.Duration // 响应超时（可选，默认10s）
	Rack                int           // 机架号
	Slot                int           // 槽位号
	Destination_TSAP    int           // 远端TSAP
	Source_TSAP         int           // 本地TSAP
	Delay_between_polls time.Duration // 轮询间隔（可选，默认0）
	MaxPacketLen        int           // 组包最大长度（可选，默认480） Smart200=240 300=240 400=960 1200=480 1500=960
	ConnectionType      int           // 连接类型（可选，默认2） 1=PG编程设备 2=OP操作面板 3=Basic
}

// S7数据包点位结构体
type S7Points_type struct {
	Area          int // 存储区 0x81=I(输入) 0x82=Q(输出) 0x83=M(标志) 0x84=DB(数据块) 0x1C=C(计数器) 0x1D=T(定时器)（snap7 编码）
	DBNumber      int // DB号
	Start         int // 字节偏移 1byte = 8bit
	Type          int // 采集数据类型 1=bool(Bool) 2=int8(SInt) 3=uint8(USInt) 4=int16(Int) 5=uint16(UInt) 6=int32(DInt) 7=uint32(UDInt) 8=int64(LInt) 9=uint64(ULInt) 10=int 11=uint 12=float32(Real) 13=float64(LReal) 14=float 15=string
	Child_Address int // 子地址（可选）
}

type Point_Config_type struct {
	Id  uint // 点位id
	Tag string

	RW_Cancel  int    // 读写方式读写方式 1：禁止； 2：只读； 3：只写； 4：读写；
	Value_Type int    // 输出类型 1：bool； 2：int8； 3：uint8； 4：int16； 5：uint16； 6：int32； 7：uint32； 8：int64； 9：uint64； 10：int； 11：uint； 12：float32； 13：float64； 14：float； 15：string；
	Conversion string // 换算表达式

	Config S7Points_type
}

type Siemens_S7 struct {
	Drive_Id uint
	Config   Config_type
	Points   []Point_Config_type

	Read_Packet_S7DataItem       []S7DataItem
	Read_Packet_S7DataItem_Point map[int][]int // 读取数据包中点位的映射 map[Read_Packet_S7DataItem的index]Points的index

	client        *Snap7Client // snap7 客户端
	closed        bool         // 连接关闭标志
	retryCount    int          // 重试计数
	negotiatedPDU int          // 连接协商后的实际 PDU 长度

	Read_External_Mappings func([]fullConfig.Value_type) error // 外部映射回调
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
		points = append(points, Point_Config_type{
			Id:         v.Id,
			Tag:        v.Tag,
			RW_Cancel:  v.RW_Cancel,
			Value_Type: v.Value_Type,
			Conversion: v.Conversion,
			Config:     point,
		})
	}
	c.Points = points

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
		points = append(points, Point_Config_type{
			Id:         v.Id,
			Tag:        v.Tag,
			RW_Cancel:  v.RW_Cancel,
			Value_Type: v.Value_Type,
			Conversion: v.Conversion,
			Config:     point,
		})
	}
	c.Points = points

	if err := c.Read_Packet(); err != nil {
		return fmt.Errorf("UpdatePoints 读取组包失败: %w", err)
	}

	log.Printf("INFO 驱动【%d】 点位动态更新完成, 点位数【%d】, 轮询包数【%d】", c.Drive_Id, len(c.Points), len(c.Read_Packet_S7DataItem))
	return nil
}

/*******************驱动连接*******************/

// packet 组包：筛选可读点位（RW_Cancel=2只读/4读写），按 Area+DBNumber 分组合并连续地址，
// 构建 Read_Packet_S7DataItem 和 Read_Packet_S7DataItem_Point
func (c *Siemens_S7) Read_Packet() error {
	// 1. 筛选可读点位，转换为 S7PackPoint
	var packPoints []S7PackPoint
	for _, p := range c.Points {
		if p.RW_Cancel != 2 && p.RW_Cancel != 4 {
			continue
		}
		byteLen := ValueTypeToByteLen(p.Config.Type) // 用采集类型（PLC 实际数据类型）决定缓冲区大小
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
			ValueType: p.Value_Type,
		})
	}

	if len(packPoints) == 0 {
		log.Printf("WARN 驱动【%d】 无有效的可读点位, 跳过组包", c.Drive_Id)
		return fmt.Errorf("无有效的可读点位")
	}

	// 2. 调用组包函数（优先使用协商后的实际 PDU，未连接时用配置值）
	pduLen := c.negotiatedPDU
	if pduLen <= 0 {
		pduLen = c.Config.MaxPacketLen
	}
	packs, err := PackS7AddressPackages(packPoints, pduLen)
	if err != nil {
		log.Printf("WARN 驱动【%d】 S7 组包失败: %v", c.Drive_Id, err)
		return fmt.Errorf("S7 组包失败: %w", err)
	}

	// 3. 构建 S7DataItem 和点位映射
	c.Read_Packet_S7DataItem = make([]S7DataItem, 0, len(packs))
	c.Read_Packet_S7DataItem_Point = make(map[int][]int)

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
			Area:     pack.Area,
			WordLen:  wordLen,
			DBNumber: dbNum,
			Start:    start,
			Amount:   amount,
			Data:     make([]byte, dataLen),
		}
		c.Read_Packet_S7DataItem = append(c.Read_Packet_S7DataItem, item)

		// 构建 DataItem index → Points index 映射
		var pointIndices []int
		var pointIds []uint
		for _, id := range pack.Ids {
			for j, p := range c.Points {
				if p.Id == id {
					pointIndices = append(pointIndices, j)
					pointIds = append(pointIds, p.Id)
					break
				}
			}
		}
		c.Read_Packet_S7DataItem_Point[i] = pointIndices

		// 诊断日志：点位 ID、DB、起始/结束字节地址、报文参数
		endByte := pack.Start + pack.ByteLen - 1
		log.Printf("DEBUG 驱动【%d】 DataItem【%d】 点位IDs=%v Area=0x%X DB=%d Start=%d End=%d Amount=%d WordLen=0x%X BufLen=%d",
			c.Drive_Id, i, pointIds, item.Area, item.DBNumber, item.Start, endByte, item.Amount, item.WordLen, len(item.Data))
	}

	return nil
}

/*******************驱动连接*******************/

// Connect 绑定外部映射回调，建立连接，启动轮询
func (c *Siemens_S7) Connect(Read_External_Mappings func([]fullConfig.Value_type) error) error {
	c.Read_External_Mappings = Read_External_Mappings
	c.closed = false // 重连时重置关闭标志

	err := c.connect()
	if err != nil {
		// 首次连接失败，后台持续重连直到成功或被关闭
		log.Printf("WARN 驱动【%d】 首次连接失败: %v, 进入重连循环", c.Drive_Id, err)
		go func() {
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				log.Printf("ERROR 驱动【%d】 重连最终失败: %v", c.Drive_Id, reconnectErr)
				c.Error_External_Mappings(reconnectErr.Error())
				return
			}
			c.polling()
		}()
		return nil
	}
	go c.polling()
	return nil
}

// connect 建立 S7 TCP 连接（使用 snap7 C 动态库）
func (c *Siemens_S7) connect() error {
	client := NewSnap7Client()

	// 设置连接类型
	connType := c.Config.ConnectionType
	if connType <= 0 {
		connType = ConnTypeOP // 默认 OP 连接
	}
	client.SetConnectionType(connType)

	// 自定义 TSAP 或 默认 ConnectTo
	if c.Config.Source_TSAP != 0 || c.Config.Destination_TSAP != 0 {
		client.SetConnectionParams(c.Config.Address, uint16(c.Config.Source_TSAP), uint16(c.Config.Destination_TSAP))
	} else {
		// ConnectTo 内部自动计算 TSAP = ConnectionType<<8 + Rack*0x20 + Slot
		// 先设置连接参数再 Connect
		client.SetConnectionParams(c.Config.Address, uint16(connType)<<8, uint16(connType)<<8+uint16(c.Config.Rack)*0x20+uint16(c.Config.Slot))
	}

	// 设置超时参数（毫秒）
	if c.Config.Connect_timeout > 0 {
		client.SetParam(ParamSendTimeout, int(c.Config.Connect_timeout.Milliseconds()))
	}
	if c.Config.Response_timeout > 0 {
		client.SetParam(ParamRecvTimeout, int(c.Config.Response_timeout.Milliseconds()))
	}

	log.Printf("INFO siemens_s7 连接参数: %+v", c.Config)
	code := client.Connect()
	if code != 0 {
		client.Destroy()
		return fmt.Errorf("siemens_s7 连接失败: %s", snap7ErrorText(code))
	}

	c.client = client
	c.closed = false

	_, pduLen := client.GetPduLength()
	c.negotiatedPDU = pduLen

	// 诊断：ReadArea 单项测试读取 DB1.DBB0~3，验证基本通信
	testBuf := make([]byte, 4)
	testResult := client.ReadArea(S7AreaDB, 1, 0, 4, S7WLByte, testBuf)
	log.Printf("DEBUG 驱动【%d】 ReadArea测试: DB1.DBB0~3, result=%d (%s), data=%v",
		c.Drive_Id, testResult, snap7ErrorText(testResult), testBuf)

	// 用协商后的实际 PDU 重组包（替换 New() 中基于配置值的初始组包）
	if pduLen > 0 {
		if err := c.Read_Packet(); err != nil {
			log.Printf("WARN 驱动【%d】 协商PDU【%d】重组包失败: %v", c.Drive_Id, pduLen, err)
		}
	}

	log.Printf("INFO 驱动【%d】 连接成功, 地址【%s】, rack【%d】, slot【%d】, PDU【%d】", c.Drive_Id, c.Config.Address, c.Config.Rack, c.Config.Slot, pduLen)
	return nil
}

// reconnect 掉线重连，按 Retry_timeout 间隔重试
func (c *Siemens_S7) reconnect() error {
	retryTimeout := c.Config.Retry_timeout
	if retryTimeout <= 0 {
		retryTimeout = 3 * time.Second
	}

	for {
		if c.closed {
			return fmt.Errorf("驱动已关闭")
		}

		log.Printf("INFO 驱动【%d】 尝试重连, 地址【%s】 ...", c.Drive_Id, c.Config.Address)

		// 关闭旧连接
		if c.client != nil {
			c.client.Disconnect()
			c.client.Destroy()
		}

		err := c.connect()
		if err == nil {
			log.Printf("INFO 驱动【%d】 重连成功", c.Drive_Id)
			return nil
		}

		// 重连期间上报所有点位错误状态
		c.Error_External_Mappings(err.Error())

		log.Printf("WARN 驱动【%d】 重连失败: %v, %s 后重试", c.Drive_Id, err, retryTimeout)

		time.Sleep(retryTimeout)
		c.retryCount++ // 重试计数
	}
}

// Close 关闭驱动连接
func (c *Siemens_S7) Close() error {
	c.closed = true
	if c.client != nil {
		c.client.Disconnect()
		c.client.Destroy()
	}
	c.Error_External_Mappings("驱动连接已关闭")
	return nil
}

// polling 轮询读取所有数据包，掉线自动重连
// 按协商 PDU 分批发送 ReadMultiVars，每批总字节（数据 + 17字节/item 开销）不超过 PDU
func (c *Siemens_S7) polling() {
	if len(c.Read_Packet_S7DataItem) == 0 {
		log.Printf("WARN 驱动【%d】 无有效轮询包, 退出轮询", c.Drive_Id)
		return
	}

	// 计算分批策略：每批总 PDU 不超过 negotiatedPDU
	maxPDU := c.negotiatedPDU
	if maxPDU <= 0 {
		maxPDU = c.Config.MaxPacketLen
	}
	if maxPDU <= 0 {
		maxPDU = s7DefaultPDUSize
	}

	type batch struct{ start, end int }
	var batches []batch
	batchStart := 0
	batchBytes := 0

	for i, item := range c.Read_Packet_S7DataItem {
		itemTotal := item.Amount + s7ItemOverhead
		if i == batchStart && itemTotal >= maxPDU {
			// 单个 item 就超 PDU，单独成批
			batches = append(batches, batch{start: i, end: i + 1})
			batchStart = i + 1
			batchBytes = 0
			continue
		}
		if batchBytes+itemTotal > maxPDU && i > batchStart {
			batches = append(batches, batch{start: batchStart, end: i})
			batchStart = i
			batchBytes = 0
		}
		batchBytes += itemTotal
	}
	if batchStart < len(c.Read_Packet_S7DataItem) {
		batches = append(batches, batch{start: batchStart, end: len(c.Read_Packet_S7DataItem)})
	}

	log.Printf("INFO 驱动【%d】 轮询启动, 包数【%d】, 分批【%d】, PDU【%d】",
		c.Drive_Id, len(c.Read_Packet_S7DataItem), len(batches), maxPDU)

	for {
		if c.closed {
			log.Printf("INFO 驱动【%d】 已关闭, 退出轮询", c.Drive_Id)
			return
		}

		// 逐批读取
		readErr := false
		for _, b := range batches {
			batchItems := c.Read_Packet_S7DataItem[b.start:b.end]
			code := c.client.ReadMultiVars(batchItems)
			if code != 0 {
				log.Printf("WARN 驱动【%d】 读取错误: %s, 尝试重连", c.Drive_Id, snap7ErrorText(code))
				readErr = true
				break
			}
			// 回写结果到原数组
			for j, item := range batchItems {
				c.Read_Packet_S7DataItem[b.start+j] = item
			}
		}

		if readErr {
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				log.Printf("ERROR 驱动【%d】 重连最终失败, err【%v】, 退出轮询", c.Drive_Id, reconnectErr)
				c.Error_External_Mappings(reconnectErr.Error())
				return
			}
			// 重连成功后重新计算分批（PDU 可能变化）
			maxPDU = c.negotiatedPDU
			if maxPDU <= 0 {
				maxPDU = s7DefaultPDUSize
			}
			batches = batches[:0]
			batchStart = 0
			batchBytes = 0
			for i, item := range c.Read_Packet_S7DataItem {
				itemTotal := item.Amount + s7ItemOverhead
				if i == batchStart && itemTotal >= maxPDU {
					batches = append(batches, batch{start: i, end: i + 1})
					batchStart = i + 1
					batchBytes = 0
					continue
				}
				if batchBytes+itemTotal > maxPDU && i > batchStart {
					batches = append(batches, batch{start: batchStart, end: i})
					batchStart = i
					batchBytes = 0
				}
				batchBytes += itemTotal
			}
			if batchStart < len(c.Read_Packet_S7DataItem) {
				batches = append(batches, batch{start: batchStart, end: len(c.Read_Packet_S7DataItem)})
			}
			continue
		}

		// 逐个 DataItem 检查结果
		totalItems := len(c.Read_Packet_S7DataItem)
		for j := 0; j < totalItems; j++ {
			if j >= len(c.Read_Packet_S7DataItem) {
				log.Printf("WARN 驱动【%d】 DataItem 索引【%d】超出范围【%d】，跳过剩余", c.Drive_Id, j, len(c.Read_Packet_S7DataItem))
				break
			}
			item := &c.Read_Packet_S7DataItem[j]
			if item.Result != 0 {
				errMsg := snap7ErrorText(item.Result)
				c.Error_External_Mappings_byPacketIndex(j, errMsg)
				continue
			}

			readList := c.analysis(j, item.Data)
			if len(readList) > 0 && c.Read_External_Mappings != nil {
				if cbErr := c.Read_External_Mappings(readList); cbErr != nil {
					log.Printf("WARN 驱动【%d】 数据回调失败, err【%v】", c.Drive_Id, cbErr)
				}
			}
		}

		time.Sleep(c.Config.Delay_between_polls)
	}
}

// analysis 解析单个 DataItem 的读取数据，返回点位值列表
func (c *Siemens_S7) analysis(packetIndex int, data []byte) []fullConfig.Value_type {
	pointIndices, ok := c.Read_Packet_S7DataItem_Point[packetIndex]
	if !ok {
		return nil
	}

	item := c.Read_Packet_S7DataItem[packetIndex]
	var readList []fullConfig.Value_type
	now := time.Now()

	for _, pIdx := range pointIndices {
		if pIdx < 0 || pIdx >= len(c.Points) {
			continue
		}
		point := c.Points[pIdx]

		byteOffset := point.Config.Start - item.Start // snap7 Start 直接是字节偏移
		byteLen := ValueTypeToByteLen(point.Config.Type)
		if byteLen <= 0 {
			log.Printf("WARN 点位【%s】 采集类型无效, 类型编码【%d】, 跳过", point.Tag, point.Config.Type)
			continue
		}
		if byteOffset < 0 || byteOffset+byteLen > len(data) {
			log.Printf("WARN 点位【%s】 字节偏移越界, offset【%d】, byteLen【%d】, dataLen【%d】", point.Tag, byteOffset, byteLen, len(data))
			continue
		}

		read := fullConfig.Value_type{
			PointId: point.Id,
			Type:    point.Value_Type,
			Msg:     "ok",
			Time:    now,
		}

		// 1. 按采集类型 Type 从字节流解析原始值（S7 固定大端序）
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
			// 2. 换算 + 类型转换
			result, err := conv.ConvertWithCalc(v, point.Config.Type, point.Value_Type, point.Conversion)
			if err != nil {
				read.Msg = fmt.Sprintf("点位【%s】 %s, 采集类型【%d】, 输出类型【%d】", point.Tag, err.Error(), point.Config.Type, point.Value_Type)
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
	readList := make([]fullConfig.Value_type, 0, len(c.Points))
	for _, point := range c.Points {
		readList = append(readList, fullConfig.Value_type{
			PointId: point.Id,
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
	pointIndices, ok := c.Read_Packet_S7DataItem_Point[packetIndex]
	if !ok {
		return nil
	}
	readList := make([]fullConfig.Value_type, 0, len(pointIndices))
	for _, pIdx := range pointIndices {
		if pIdx < 0 || pIdx >= len(c.Points) {
			continue
		}
		point := c.Points[pIdx]
		readList = append(readList, fullConfig.Value_type{
			PointId: point.Id,
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
func (c *Siemens_S7) Write(values []fullConfig.Value_type) error {
	// 1. 校验每个写入值对应的点位是否存在且允许写入
	tagPointsMap := make(map[uint]fullConfig.Value_type)
	for _, v := range values {
		tagPointsMap[v.PointId] = v
	}
	for _, p := range c.Points {
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
	for i, p := range c.Points {
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
			p := c.Points[idx]
			byteLen := ValueTypeToByteLen(p.Config.Type)
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
		// 优先使用协商后的实际 PDU
		pduLen := c.negotiatedPDU
		if pduLen <= 0 {
			pduLen = c.Config.MaxPacketLen
		}
		packs, err := PackS7AddressPackages(packPoints, pduLen)
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
				Area: area, WordLen: wordLen, DBNumber: dbNum,
				Start: start, Amount: amount,
				Data: make([]byte, dataLen),
			}
			// 构建本包涉及的点位索引
			var pointIndices []int
			for _, id := range pack.Ids {
				for j, p := range c.Points {
					if p.Id == id {
						if _, exists := tagPointsMap[p.Id]; exists {
							pointIndices = append(pointIndices, j)
						}
						break
					}
				}
			}
			// 检查本包是否包含 bool 点位
			hasBool := false
			for _, idx := range pointIndices {
				if c.Points[idx].Config.Type == conv.TYPE_BOOL {
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

	// 3. 分离 bool 包（需要先读）和直接写入包
	var boolItems []S7DataItem
	var boolIndices []int // boolItems 在 groups 中的索引
	var directItems []S7DataItem
	var directIndices []int

	for i, g := range groups {
		if g.hasBool {
			boolItems = append(boolItems, g.item)
			boolIndices = append(boolIndices, i)
		} else {
			directItems = append(directItems, g.item)
			directIndices = append(directIndices, i)
		}
	}

	// 4. bool 包：AGReadMulti 先读取当前值（位操作需要保留同字节其他位）
	if len(boolItems) > 0 {
		if len(boolItems) > 20 {
			for start := 0; start < len(boolItems); start += 20 {
				end := start + 20
				if end > len(boolItems) {
					end = len(boolItems)
				}
				batch := boolItems[start:end]
				if code := c.client.ReadMultiVars(batch); code != 0 {
					return fmt.Errorf("ReadMultiVars 批量读取失败 batch=%d: %s", start/20, snap7ErrorText(code))
				}
			}
		} else {
			if code := c.client.ReadMultiVars(boolItems); code != 0 {
				return fmt.Errorf("ReadMultiVars 批量读取失败: %s", snap7ErrorText(code))
			}
		}
		// 回写读取结果到 groups
		for i, gIdx := range boolIndices {
			groups[gIdx].item = boolItems[i]
		}
	}

	// 5. 逐包更新缓冲区
	for i := range groups {
		if err := writeUpdateBuffer(&groups[i].item, groups[i].pointIndices, c.Points, tagPointsMap); err != nil {
			log.Printf("ERROR 写入准备失败, Area【%d】, DB【%d】, Start【%d】, err【%v】", groups[i].item.Area, groups[i].item.DBNumber, groups[i].item.Start, err)
		}
	}

	// 6. AGWriteMulti 批量写入 PLC（bool 包 + 直接包分别写入）
	allItems := make([]S7DataItem, 0, len(groups))
	for _, g := range groups {
		allItems = append(allItems, g.item)
	}
	if len(allItems) > 20 {
		for start := 0; start < len(allItems); start += 20 {
			end := start + 20
			if end > len(allItems) {
				end = len(allItems)
			}
			batch := allItems[start:end]
			if code := c.client.WriteMultiVars(batch); code != 0 {
				return fmt.Errorf("WriteMultiVars 批量写入失败 batch=%d: %s", start/20, snap7ErrorText(code))
			}
		}
		return nil
	}
	if code := c.client.WriteMultiVars(allItems); code != 0 {
		return fmt.Errorf("WriteMultiVars 批量写入失败: %s", snap7ErrorText(code))
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
func writeUpdateBuffer(item *S7DataItem, pointIndices []int, points []Point_Config_type, tagPointsMap map[uint]fullConfig.Value_type) error {
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

		byteOffset := point.Config.Start - item.Start // snap7 Start 直接是字节偏移
		byteLen := ValueTypeToByteLen(point.Config.Type)
		if byteOffset < 0 || byteOffset+byteLen > len(item.Data) {
			return fmt.Errorf("点位【%s】 字节偏移越界, offset【%d】, byteLen【%d】, bufLen【%d】", point.Tag, byteOffset, byteLen, len(item.Data))
		}

		// 反向换算 + 类型转换：Value_Type → Type（采集类型）
		convertedValue, err := conv.ConvertReverse(v.Value, point.Value_Type, point.Config.Type, point.Conversion)
		if err != nil {
			return fmt.Errorf("点位【%s】 写入值类型转换失败, err【%v】", point.Tag, err)
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
			// 非 bool 类型：用 conv 转字节（S7 大端序）并写入缓冲区
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
				return fmt.Errorf("点位【%s】 不支持的写入类型【%d】", point.Tag, point.Config.Type)
			}
		}
	}

	return nil
}
