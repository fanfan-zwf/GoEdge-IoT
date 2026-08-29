package siemenss7

import (
	"fmt"
	"log"
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"main/db/mysql"
	"time"

	"github.com/robinson/gos7"
)

type Config_type struct {
	Address             string        // 地址
	Retry_timeout       time.Duration // 重试间隔（可选，默认3s）
	Connect_timeout     time.Duration // 连接超时（可选，默认10s）
	Response_timeout    time.Duration // 响应超时（可选，默认10s）
	Rack                int           // 机架号
	Slot                int           // 槽位号
	Delay_between_polls time.Duration // 轮询间隔（可选，默认0）
	MaxPacketLen        int           // 组包最大长度（可选，默认480） Smart200=240 300=240 400=960 1200=480 1500=960
	ConnectionType      int           // 连接类型（可选，默认2） 1=PG编程设备 2=OP操作面板 3=Basic
}

type Points_type struct {
	Id            uint // 点位id
	Area          int  // 存储区 0x81=I(输入) 0x82=Q(输出) 0x83=M(标志) 0x84=DB(数据块) 0x1C=T(定时器) 0x1B=C(计数器)
	DBNumber      int  // DB号
	Start         int  // 字节偏移
	Type          int  // 采集数据类型 1=bool(Bool) 2=int8(SInt) 3=uint8(USInt) 4=int16(Int) 5=uint16(UInt) 6=int32(DInt) 7=uint32(UDInt) 8=int64(LInt) 9=uint64(ULInt) 10=int 11=uint 12=float32(Real) 13=float64(LReal) 14=float 15=string
	Child_Address int  // 子地址（可选）

	RW_Cancel  int // 读写方式读写方式 1：禁止； 2：只读； 3：只写； 4：读写；
	Value_Type int // 输出类型 1：bool； 2：int8； 3：uint8； 4：int16； 5：uint16； 6：int32； 7：uint32； 8：int64； 9：uint64； 10：int； 11：uint； 12：float32； 13：float64； 14：float； 15：string；
}

type Siemens_S7 struct {
	Config Config_type
	Points []Points_type

	Read_Packet_S7DataItem       []gos7.S7DataItem
	Read_Packet_S7DataItem_Point map[int][]int // 读取数据包中点位的映射 map[Read_Packet_S7DataItem的index]Points的index

	handler *gos7.TCPClientHandler // TCP 连接处理器
	conn    gos7.Client            // S7 客户端
	closed  bool                   // 连接关闭标志

	Read_External_Mappings func([]fullConfig.Value_type) error // 外部映射回调
}

/*******************驱动初始化*******************/

// New 解析驱动配置和点位配置，构建读取数据包
func (c *Siemens_S7) New(Drive mysql.CollectorGet_Drive_Config_type, Points []mysql.CollectorGet_Point_Config_type) (err error) {

	// 1. 解析驱动配置字符串
	c.Config, err = Drive_Config_Switch(Drive.Config)
	if err != nil {
		return fmt.Errorf("解析驱动配置失败: %w", err)
	}

	// 2. 解析点位配置
	var points []Points_type
	for _, v := range Points {
		point, err := Point_Config_Switch(v.Config)
		if err != nil {
			log.Printf("WARN 点位 id=%d 配置解析失败，跳过: %v", v.Id, err)
			continue
		}
		point.Id = v.Id
		point.RW_Cancel = v.RW_Cancel
		point.Value_Type = v.Value_Type
		points = append(points, point)
	}
	c.Points = points

	// 3. 组包
	if err := c.Read_Packet(); err != nil {
		return fmt.Errorf("组包失败: %w", err)
	}

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
		byteLen := ValueTypeToByteLen(p.Value_Type)
		if byteLen <= 0 {
			log.Printf("WARN siemens_s7: 无效 Value_Type=%d 点位 id=%d，跳过", p.Value_Type, p.Id)
			continue
		}
		packPoints = append(packPoints, S7PackPoint{
			Id:        p.Id,
			Area:      p.Area,
			DBNumber:  p.DBNumber,
			Start:     p.Start,
			ByteLen:   byteLen,
			ValueType: p.Value_Type,
		})
	}

	if len(packPoints) == 0 {
		log.Printf("WARN siemens_s7: 无有效的可读点位，跳过组包")
		return fmt.Errorf("无有效的可读点位")
	}

	// 2. 调用组包函数
	packs, err := PackS7AddressPackages(packPoints, c.Config.MaxPacketLen)
	if err != nil {
		log.Printf("WARN siemens_s7: S7 组包失败: %v", err)
		return fmt.Errorf("S7 组包失败: %w", err)
	}

	// 3. 构建 S7DataItem 和点位映射
	c.Read_Packet_S7DataItem = make([]gos7.S7DataItem, 0, len(packs))
	c.Read_Packet_S7DataItem_Point = make(map[int][]int)

	for i, pack := range packs {
		item := gos7.S7DataItem{
			Area:     pack.Area,
			WordLen:  0x02, // S7WLByte: 按字节读取
			DBNumber: pack.DBNumber,
			Start:    pack.Start,
			Amount:   pack.ByteLen,
			Data:     make([]byte, pack.ByteLen),
		}
		c.Read_Packet_S7DataItem = append(c.Read_Packet_S7DataItem, item)

		// 构建 DataItem index → Points index 映射
		var pointIndices []int
		for _, id := range pack.Ids {
			for j, p := range c.Points {
				if p.Id == id {
					pointIndices = append(pointIndices, j)
					break
				}
			}
		}
		c.Read_Packet_S7DataItem_Point[i] = pointIndices
	}

	return nil
}

/*******************驱动连接*******************/

// Connect 绑定外部映射回调，建立连接，启动轮询
func (c *Siemens_S7) Connect(Read_External_Mappings func([]fullConfig.Value_type) error) error {
	c.Read_External_Mappings = Read_External_Mappings

	err := c.connect()
	if err != nil {
		return err
	}
	go c.polling()
	return nil
}

// connect 建立 S7 TCP 连接
func (c *Siemens_S7) connect() error {
	connType := c.Config.ConnectionType
	if connType <= 0 {
		connType = 2 // 默认 OP 连接
	}
	handler := gos7.NewTCPClientHandlerWithConnectType(c.Config.Address, c.Config.Rack, c.Config.Slot, connType)

	// 连接超时（默认 10s）
	if c.Config.Connect_timeout > 0 {
		handler.Timeout = c.Config.Connect_timeout
	}
	// 响应超时（默认 10s）
	if c.Config.Response_timeout > 0 {
		handler.IdleTimeout = c.Config.Response_timeout
	}

	err := handler.Connect()
	if err != nil {
		c.Error_External_Mappings(err.Error())
		return fmt.Errorf("siemens_s7 连接失败: %w", err)
	}

	c.handler = handler
	c.conn = gos7.NewClient(handler)
	c.closed = false

	log.Printf("siemens_s7 连接成功 地址:%s rack:%d slot:%d", c.Config.Address, c.Config.Rack, c.Config.Slot)
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

		log.Printf("siemens_s7 尝试重连 地址:%s ...", c.Config.Address)

		// 关闭旧连接
		if c.handler != nil {
			c.handler.Close()
		}

		err := c.connect()
		if err == nil {
			log.Printf("siemens_s7 重连成功")
			return nil
		}

		log.Printf("siemens_s7 重连失败: %v，%s 后重试", err, retryTimeout)
		time.Sleep(retryTimeout)
	}
}

// Close 关闭驱动连接
func (c *Siemens_S7) Close() error {
	c.closed = true
	if c.handler != nil {
		c.handler.Close()
	}
	c.Error_External_Mappings("驱动连接已关闭")
	return nil
}

// polling 轮询读取所有数据包，掉线自动重连
func (c *Siemens_S7) polling() {
	if len(c.Read_Packet_S7DataItem) == 0 {
		log.Printf("WARN siemens_s7: 无有效轮询包，退出轮询")
		return
	}

	var i int
	for {
		if c.closed {
			log.Printf("INFO siemens_s7: 驱动已关闭，退出轮询")
			return
		}

		if i < 0 || i >= len(c.Read_Packet_S7DataItem) {
			i = 0
		}

		itemCount := len(c.Read_Packet_S7DataItem)
		err := c.conn.AGReadMulti(c.Read_Packet_S7DataItem, itemCount)

		if err != nil {
			log.Printf("WARN siemens_s7: 读取错误: %v，尝试重连", err)

			// 重连失败 → 上报所有点位错误，退出轮询
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				log.Printf("ERROR siemens_s7: 重连最终失败: %v，退出轮询", reconnectErr)
				c.Error_External_Mappings(reconnectErr.Error())
				return
			}
			continue // 重连成功，继续轮询
		}

		// 逐个 DataItem 检查结果
		for j, item := range c.Read_Packet_S7DataItem {
			if item.Error != "" {
				log.Printf("WARN siemens_s7: DataItem[%d] 读取失败: %s", j, item.Error)
				c.Error_External_Mappings_byPacketIndex(j, item.Error)
				continue
			}

			// 解析数据
			readList := c.analysis(j, item.Data)
			if len(readList) > 0 && c.Read_External_Mappings != nil {
				if cbErr := c.Read_External_Mappings(readList); cbErr != nil {
					log.Printf("WARN siemens_s7: 数据回调失败: %v", cbErr)
				}
			}
		}

		time.Sleep(c.Config.Delay_between_polls) // 轮询间隔
		i++
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

		byteOffset := point.Start - item.Start
		byteLen := ValueTypeToByteLen(point.Type)
		if byteLen <= 0 {
			log.Printf("WARN siemens_s7: 点位 id=%d 采集类型 Type=%d 无效，跳过", point.Id, point.Type)
			continue
		}
		if byteOffset < 0 || byteOffset+byteLen > len(data) {
			log.Printf("WARN siemens_s7: 点位 id=%d 字节偏移越界 offset=%d byteLen=%d dataLen=%d", point.Id, byteOffset, byteLen, len(data))
			continue
		}

		read := fullConfig.Value_type{
			PointId: point.Id,
			Type:    ValueTypeIntToString(point.Value_Type),
			Msg:     "ok",
			Time:    now,
		}

		// 1. 按采集类型 Type 从字节流解析原始值（S7 固定大端序）
		v, ok := parseS7Value(point.Type, point.Child_Address, data[byteOffset:])
		if !ok {
			read.Msg = fmt.Sprintf("解析失败: Type=%d", point.Type)
		}

		if ok {
			// 2. 采集类型 → 输出类型 转换
			read.Value, ok = byte_util.ConvertType(v, ValueTypeIntToString(point.Type), ValueTypeIntToString(point.Value_Type))
			if !ok {
				read.Msg = fmt.Sprintf("类型转换失败: 采集Type=%d 输出Value_Type=%d 实际%T", point.Type, point.Value_Type, v)
			}
		}

		readList = append(readList, read)
	}

	return readList
}

// parseS7Value 按采集类型 Type 从 S7 字节流解析原始值
// S7 协议数据固定大端序：2字节=BA，4字节=ABCD，8字节=ABCDEFGH
// bool 类型通过 Child_Address 提取对应位
func parseS7Value(valueType int, childAddress int, data []byte) (any, bool) {
	switch valueType {
	case 1: // bool → 提取 data[0] 的第 childAddress 位
		if len(data) < 1 {
			return nil, false
		}
		return (data[0] & (1 << uint(childAddress))) != 0, true

	case 2: // int8
		if len(data) < 1 {
			return nil, false
		}
		return int8(data[0]), true

	case 3: // uint8
		if len(data) < 1 {
			return nil, false
		}
		return data[0], true

	case 4: // int16 → 大端 BA
		if len(data) < 2 {
			return nil, false
		}
		return byte_util.BytesToInt16(data[:2], byte_util.BA)[0], true

	case 5: // uint16 → 大端 BA
		if len(data) < 2 {
			return nil, false
		}
		return byte_util.BytesToUint16(data[:2], byte_util.BA)[0], true

	case 6: // int32 → 大端 ABCD
		if len(data) < 4 {
			return nil, false
		}
		return byte_util.BytesToInt32(data[:4], byte_util.ABCD)[0], true

	case 7: // uint32 → 大端 ABCD
		if len(data) < 4 {
			return nil, false
		}
		return byte_util.BytesToUint32(data[:4], byte_util.ABCD)[0], true

	case 8: // int64 → 大端 ABCDEFGH
		if len(data) < 8 {
			return nil, false
		}
		return byte_util.BytesToInt64(data[:8], byte_util.ABCDEFGH)[0], true

	case 9: // uint64 → 大端 ABCDEFGH
		if len(data) < 8 {
			return nil, false
		}
		return byte_util.BytesToUint64(data[:8], byte_util.ABCDEFGH)[0], true

	case 10: // int → 按 int64 处理
		if len(data) < 8 {
			return nil, false
		}
		return int(byte_util.BytesToInt64(data[:8], byte_util.ABCDEFGH)[0]), true

	case 11: // uint → 按 uint64 处理
		if len(data) < 8 {
			return nil, false
		}
		return uint(byte_util.BytesToUint64(data[:8], byte_util.ABCDEFGH)[0]), true

	case 12: // float32 → 大端 ABCD
		if len(data) < 4 {
			return nil, false
		}
		return byte_util.BytesToFloat32(data[:4], byte_util.ABCD)[0], true

	case 13: // float64 → 大端 ABCDEFGH
		if len(data) < 8 {
			return nil, false
		}
		return byte_util.BytesToFloat64(data[:8], byte_util.ABCDEFGH)[0], true

	case 14: // float → 按 float32 处理
		if len(data) < 4 {
			return nil, false
		}
		return byte_util.BytesToFloat32(data[:4], byte_util.ABCD)[0], true

	case 15: // string → 截取到第一个 0 字节
		end := len(data)
		for i, b := range data {
			if b == 0 {
				end = i
				break
			}
		}
		return string(data[:end]), true

	default:
		return nil, false
	}
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
			Type:    ValueTypeIntToString(point.Value_Type),
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
			Type:    ValueTypeIntToString(point.Value_Type),
			Msg:     msg,
			Time:    time.Now(),
		})
	}
	return c.Read_External_Mappings(readList)
}

// ValueTypeIntToString 将 Value_Type 整数编码转换为字符串类型名
func ValueTypeIntToString(vt int) string {
	switch vt {
	case 1:
		return "bool"
	case 2:
		return "int8"
	case 3:
		return "uint8"
	case 4:
		return "int16"
	case 5:
		return "uint16"
	case 6:
		return "int32"
	case 7:
		return "uint32"
	case 8:
		return "int64"
	case 9:
		return "uint64"
	case 10:
		return "int"
	case 11:
		return "uint"
	case 12:
		return "float32"
	case 13:
		return "float64"
	case 14:
		return "float"
	case 15:
		return "string"
	default:
		return fmt.Sprintf("unknown(%d)", vt)
	}
}
