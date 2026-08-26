/*
* 日期: 2025.5.13 PM17:26
* 作者: 范范zwf
* 作用: Connect驱动
 */

package Modbus_Tcp

import (
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/mysql"
	"sync"

	"fmt"
	"log"

	"time"

	modbus "github.com/things-go/go-modbus"
)

// 类型字节数量输出
var Type_byte = map[string]uint16{
	"bool":    1,
	"int16":   1,
	"uint16":  1,
	"int32":   2,
	"uint32":  2,
	"float32": 2,
}

/*******************驱动配置*******************/

type Config_type struct {
	Address             string        // 地址
	Retry_timeout       time.Duration // 重试间隔（可选，默认3000）
	Connect_timeout     time.Duration // 连接超时（可选，默认3000）
	Response_timeout    time.Duration // 响应超时（可选，默认180000)
	Delay_between_polls time.Duration // 轮询时间（可选，默认1000）
	Packet_max          uint8         // 组包字节个数

	Write_Coils_Function    uint8 // 写线圈功能码
	Write_Register_Function uint8 // 写寄存器功能码
}

type Points_type struct {
	SlaveID       uint8  // 从机地址
	Function      uint8  // Modbus功能码（如3=读保持寄存器）
	Address       uint16 // 寄存器地址
	Type          string // 数据类型（bool/int8/float32等）
	Child_Address uint8  // 子地址（可选）
	Byte_Order    int    // 字节序（如"ABCD"表示大端）
}

// 值输出
// type value_array_type struct {
// 	Id         uint   // 点位id
// 	Comments   string // 状态
// 	Value_Type string // 值类型

// 	Value interface{} // 值
// 	Time  string      // 时间戳
// }

// ValueTypeIntToString 将数据库中的 int 类型编码转换为 string 类型名称
// 1:bool 2:int8 3:uint8 4:int16 5:uint16 6:int32 7:uint32 8:int64 9:uint64 10:int 11:uint 12:float32 13:float64 14:float 15:string
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

// mysql存储结构体
type Drive_Config_type struct {
	Id     uint   // 驱动id
	Type   string // 驱动类型
	Config Config_type
}

type Point_Config_type struct {
	Id         uint
	RW_Cancel  int    // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type string // 输出类型
	Config     Points_type
}

// 组包
type Packet_type struct {
	SlaveID        uint8  // 设备id
	Function       uint8  // 功能码
	Start_Address  uint16 // 开始地址
	Number_Address uint16 // 地址数量
	Ids            []uint // 这个包的点位
}

/*******************驱动连接*******************/

type Read_Points_type struct {
	SlaveID  uint8  // 设备id
	Function uint8  // Modbus功能码（如3=读保持寄存器）
	Address  uint16 // 寄存器地址
	Number   uint16 // 寄存器地址
}

// 定义一个结构体
type Modbus_Tcp struct {
	Drive  Drive_Config_type   // 通信参数结构体
	Points []Point_Config_type // 点位结构体

	conn          *modbus.Client // tcp连接实例
	conn_err      error          // 连接状态
	first_connect bool           // 首次连接

	read_points    []Read_Points_type // 读取结构体
	packets        []Packet_type      // 组包格式
	Esc_collection chan bool
	closed         bool // 连接关闭标志

	Tag_Pointsindex_Map map[uint]int // tag点位index索引

	Read_External_Mappings func([]fullConfig.Value_type) error
	Write_value_mu         sync.Mutex
}

// 定义接口
type Connect_interface interface {
	New() error     // 初始化
	Connect() error // 开始连接
	Close() error   // 停止连接
}

type Packet_df struct {
	SlaveID  uint8 // 设备id
	Function uint8 // 功能码
}

func (c *Modbus_Tcp) New(Drive mysql.Drive_Config_type, Points []mysql.CollectorGet_Point_Config_type) (err error) {

	// 解析驱动配置字符串格式: IP;Port;RetryTimeout;ConnectTimeout;ResponseTimeout;DelayBetweenPolls;PacketMax
	c.Drive.Config, err = Drive_Config_Switch(Drive.Config)
	if err != nil {
		return fmt.Errorf("解析驱动配置失败: %w", err)
	}

	// 设置其他字段
	c.Drive.Id = Drive.Id
	c.Drive.Type = Drive.Type

	c.Tag_Pointsindex_Map = make(map[uint]int, len(Points))

	var points []Point_Config_type
	for _, v := range Points {
		point, err := Point_Config_Switch(v.Config)
		if err != nil {
			log.Printf("WARN 点位 id=%d 配置解析失败，跳过: %v", v.Id, err)
			continue
		}
		// Value_Type 为 0 时，使用采集类型作为默认输出类型
		outputType := ValueTypeIntToString(v.Value_Type)
		if v.Value_Type == 0 {
			outputType = point.Type
			log.Printf("WARN 点位 id=%d 输出类型未配置，使用采集类型 %s 作为默认值", v.Id, outputType)
		}
		points = append(points, Point_Config_type{
			Id:         v.Id,
			Config:     point,
			RW_Cancel:  v.RW_Cancel,
			Value_Type: outputType,
		})
		c.Tag_Pointsindex_Map[v.Id] = len(points) - 1 // 建立tag到index的映射
	}
	c.Points = points

	c.packets, err = c.packet(c.Points, map[int]bool{2: true, 4: true}) // 2=只读(R) 4=读写(R/W)
	if err != nil {
		return fmt.Errorf("组包失败: %w", err)
	}

	return nil
}

func (c *Modbus_Tcp) id_points_index(id uint) (p Point_Config_type, err error) {
	index, exists := c.Tag_Pointsindex_Map[id]
	if !exists {
		err = fmt.Errorf("ERROR 点位不存在:  c.id_points_index:%+ v   点位id: %d", c.Tag_Pointsindex_Map, id)
		return
	}

	if index < 0 || index >= len(c.Points) {
		err = fmt.Errorf("ERROR 点位下标越界, index: %d, 切片长度: %d", index, len(c.Points))
		return
	}

	p = c.Points[index]
	return
}
func (c *Modbus_Tcp) packet(Points []Point_Config_type, RW_Cancel map[int]bool) (Packets []Packet_type, err error) {
	// 1️⃣ 初始化 map（必须！否则 panic）
	pointMap := make(map[Packet_df][]PackAddressPackages_Point_type)

	// 2️⃣ 遍历点位，按 SlaveID + Function 分组
	for _, point := range Points {

		if !RW_Cancel[point.RW_Cancel] {
			continue
		}

		// 构建 key
		key := Packet_df{
			SlaveID:  point.Config.SlaveID,
			Function: point.Config.Function,
		}

		len, exist := Type_byte[point.Config.Type]
		if !exist {
			log.Printf("ERROR modbus_tcp: 无效类型:%s  点位Id:%d", point.Config.Type, point.Id)
			continue
		}

		// 加入分组
		pointMap[key] = append(pointMap[key], PackAddressPackages_Point_type{
			Id:        point.Id,
			StartAddr: point.Config.Address,
			DataLen:   len,
		})
	}

	for key, value := range pointMap {
		packa, err := PackAddressPackages(value, uint16(c.Drive.Config.Packet_max))
		if err != nil {
			log.Printf("ERROR modbus_tcp: 组包错误:%v", err)
			continue
		}

		for _, v := range packa {
			// ✅ 修复：必须 append 到局部变量 Packets
			Packets = append(Packets, Packet_type{
				SlaveID:        key.SlaveID,
				Function:       key.Function,
				Start_Address:  v.StartAddr,
				Number_Address: v.DataLen,
				Ids:            v.Id,
			})
		}
	}

	return
}

// 开始连接外部映射
func (c *Modbus_Tcp) Connect(Read_External_Mappings func([]fullConfig.Value_type) error) error {
	c.Read_External_Mappings = Read_External_Mappings

	err := c.connect()
	if err != nil {
		return err
	}
	go c.polling()
	return nil
}

// 开始连接
func (c *Modbus_Tcp) connect() error {

	// ---------- 2. 创建客户端，绑定所有时间参数 ----------
	p := modbus.NewTCPClientProvider(c.Drive.Config.Address, modbus.WithTCPTimeout(c.Drive.Config.Connect_timeout))

	client := modbus.NewClient(p)

	c.conn = &client

	c.conn_err = client.Connect()
	if c.conn_err != nil {
		c.Error_External_Mappings(c.conn_err.Error())
		return c.conn_err
	}

	log.Printf("modbus_tcp 驱动:%d 连接成功 地址:%s", c.Drive.Id, c.Drive.Config.Address)
	return nil
}

// 关闭连接
func (c *Modbus_Tcp) Close() error {
	c.closed = true
	if c.conn != nil {
		(*c.conn).Close()
	}
	c.Error_External_Mappings("驱动连接已关闭")
	return nil
}

func (c *Modbus_Tcp) Error_External_Mappings(msg string) error {
	if c.Read_External_Mappings == nil {
		return nil
	}
	read_list := make([]fullConfig.Value_type, 0, len(c.Points))
	for _, point := range c.Points {
		read_list = append(read_list, fullConfig.Value_type{
			DeviceId: Init.Config.APP.Uuid, // 设备id
			PointId:  point.Id,             // 点位id
			Type:     point.Value_Type,     // 输出类型
			Msg:      msg,                  // 状态信息
			Time:     time.Now(),           // 读取时间
		})
	}
	return c.Read_External_Mappings(read_list)
}

func (c *Modbus_Tcp) Error_External_Mappings_list(ids []uint, msg string) error {
	if c.Read_External_Mappings == nil {
		return nil
	}
	read_list := make([]fullConfig.Value_type, 0, len(ids))
	for _, id := range ids {
		cfg, err := c.id_points_index(id)
		if err != nil {
			log.Printf("WARN 点位 id=%d 索引查找失败，跳过: %v", id, err)
			continue
		}
		read_list = append(read_list, fullConfig.Value_type{
			DeviceId: Init.Config.APP.Uuid, // 设备id
			PointId:  id,
			Type:     cfg.Value_Type,
			Msg:      msg,
			Time:     time.Now(),
		})
	}
	return c.Read_External_Mappings(read_list)
}

// 位操作（1/0 开关量）
// ReadCoils → 01 读线圈
// ReadDiscreteInputs → 02 读离散输入
// WriteSingleCoil → 05 写单个线圈
// WriteMultipleCoils → 15 写多个线圈
// 16 位寄存器（数值）
// ReadInputRegisters → 04 读输入寄存器
// ReadHoldingRegisters → 03 读保持寄存器 ✅你用这个
// WriteSingleRegister → 06 写单个寄存器
// WriteMultipleRegisters → 16 写多个寄存器

// 读取外部映射
func (c *Modbus_Tcp) Collection_Allback() error {

	return nil
}

func (c *Modbus_Tcp) analysis(packet Packet_type, results []byte) ([]fullConfig.Value_type, error) {
	var read_list []fullConfig.Value_type
	now := time.Now()
	for _, id := range packet.Ids {
		var read fullConfig.Value_type
		cfg, err := c.id_points_index(id)
		if err != nil {
			log.Printf("WARN 点位 id=%d 索引查找失败，跳过: %v", id, err)
			continue
		}

		read.Time = now
		read.PointId = id
		read.Type = cfg.Value_Type
		read.Msg = "ok"

		byte_index := int(cfg.Config.Address - packet.Start_Address)

		var v any
		switch {
		case cfg.Config.Type == "bool" && (packet.Function == 1 || packet.Function == 2):
			v = byte_util.Get_list_index(
				byte_util.BytesToBool([]byte{byte_util.Get_list_index(results, byte_index/8, 1)[0]}),
				byte_index%8, 1)[0]
		case cfg.Config.Type == "bool" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToBool([]byte{byte_util.Get_list_index(results, byte_index/16, 1)[0]}),
				int(cfg.Config.Child_Address), 1)[0]
		case cfg.Config.Type == "uint16" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToUint16(
					byte_util.Get_list_index(results, byte_index*2, 2),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "int16" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToInt16(
					byte_util.Get_list_index(results, byte_index*2, 2),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "uint32" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToUint32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "int32" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToInt32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "float32" && (packet.Function == 3 || packet.Function == 4):
			v = byte_util.Get_list_index(
				byte_util.BytesToFloat32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		default:
			read.Msg = fmt.Sprintf("不支持的配置类型: 点位id=%d, 类型=%s", id, cfg.Value_Type)
		}
		var ok bool
		read.Value, ok = byte_util.ConvertType(v, cfg.Config.Type, cfg.Value_Type)
		if !ok {
			log.Printf("WARN 点位 id=%d 值类型转换失败, 采集类型=%s, 输出类型=%s, 实际类型=%T, 跳过",
				id, cfg.Config.Type, cfg.Value_Type, v)
			continue
		}
		read.DeviceId = Init.Config.APP.Label // 设备id
		read_list = append(read_list, read)

	}
	return read_list, nil
}

func (c *Modbus_Tcp) polling() {
	if len(c.packets) == 0 {
		log.Printf("WARN 驱动:%d 无有效轮询包，退出轮询", c.Drive.Id)

		return
	}

	var i int
	for {
		if c.closed {
			log.Printf("INFO 驱动:%d 连接已关闭，退出轮询", c.Drive.Id)
			return
		}

		if i < 0 || i >= len(c.packets) {
			i = 0 // 轮询完一个周期后重置索引
		}
		packet := c.packets[i]
		i++

		time.Sleep(c.Drive.Config.Delay_between_polls) // 轮询间隔

		var (
			byte_list []byte
			err       error
		)
		switch packet.Function {
		case 1:
			byte_list, err = (*c.conn).ReadCoils(packet.SlaveID, packet.Start_Address, packet.Number_Address)
		case 2:
			byte_list, err = (*c.conn).ReadDiscreteInputs(packet.SlaveID, packet.Start_Address, packet.Number_Address)
		case 3:
			byte_list, err = (*c.conn).ReadHoldingRegistersBytes(packet.SlaveID, packet.Start_Address, packet.Number_Address)
		case 4:
			byte_list, err = (*c.conn).ReadInputRegistersBytes(packet.SlaveID, packet.Start_Address, packet.Number_Address)
		default:
			c.Error_External_Mappings_list(packet.Ids, "Unknown function code")
			continue
		}

		if err != nil {
			log.Printf("WARN 设备id:%d 包 id=%d 读取错误: %v", c.Drive.Id, packet.Ids[0], err)
			c.Error_External_Mappings_list(packet.Ids, err.Error())
			continue
		}

		read_list, err := c.analysis(packet, byte_list)
		if err != nil {
			log.Printf("WARN 设备id:%d 分析错误: %v", c.Drive.Id, err)
			c.Error_External_Mappings_list(packet.Ids, err.Error())
			continue
		}

		// 外部映射
		if c.Read_External_Mappings != nil {
			if err := c.Read_External_Mappings(read_list); err != nil {
				log.Printf("WARN 设备id:%d 数据回调失败: %v", c.Drive.Id, err)
			}
		}

	}

}

// 写入组包
func (c *Modbus_Tcp) write_packet(packet Packet_type, tag_points_map map[uint]fullConfig.Value_type) error {

	bool_value_address := make(map[uint16]bool) // 线圈地址与值的映射

	now := time.Now()
	var byte_list []byte
	for _, id := range packet.Ids {
		var cfg Point_Config_type
		cfg, err := c.id_points_index(id)
		if err != nil {
			return fmt.Errorf("驱动 id=%d %w", c.Drive.Id, err)
		}
		v, exists := tag_points_map[id]
		if !exists {
			return fmt.Errorf("写入值不存在, 点位id: %d", id)
		}

		// 时间确认
		if !v.Time.IsZero() {
			duration := now.Sub(v.Time)
			if duration > 5*time.Second {
				return fmt.Errorf("写入值时间间隔过长, 点位id=%d, 间隔=%s", id, duration)
			}
		}

		// 类型确认
		if v.Type != cfg.Value_Type {
			return fmt.Errorf("配置类型与写入值类型不匹配, 点位id=%d, 配置类型=%s, 值类型=%s", id, cfg.Value_Type, v.Type)
		}

		index := cfg.Config.Address - packet.Start_Address // 计算相对地址索引

		var ok bool
		v.Value, ok = byte_util.ConvertType(v.Value, cfg.Value_Type, cfg.Config.Type)
		if !ok {
			return fmt.Errorf("写入值类型转换失败, 点位id=%d, 配置类型=%s, 值类型=%T", id, cfg.Value_Type, v.Value)
		}

		switch {
		case cfg.Config.Type == "bool" && packet.Function == 1:
			boolVal, ok := v.Value.(bool)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 bool, 实际 %T", id, v.Value)
			}
			a := byte_util.Get_list_index(byte_list, int(index)/8, 1)
			b := byte_util.BytesToBool(a)
			b[index] = boolVal
			rb := byte_util.BoolToBytes(b)
			byte_util.Update_List_Slice(&byte_list, int(index), rb)
		case cfg.Config.Type == "bool" && packet.Function == 3:
			boolVal, ok := v.Value.(bool)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 bool, 实际 %T", id, v.Value)
			}
			if !bool_value_address[cfg.Config.Address] {
				byte_list, err = (*c.conn).ReadHoldingRegistersBytes(cfg.Config.SlaveID, cfg.Config.Address, 1)
				if err != nil {
					return fmt.Errorf("点位 id=%d 读取保持寄存器失败: %w", id, err)
				}
				byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(byte_list, 0, 2))
			}
			a := byte_util.Get_list_index(byte_list, int(index)*2, 2)
			bool_list := byte_util.Get_list_index(byte_util.BytesToBool(a), 0, 16)
			if cfg.Config.Child_Address > 15 {
				return fmt.Errorf("点位 id=%d 子地址超出范围: child_address=%d, 最大15", id, cfg.Config.Child_Address)
			}
			bool_list[cfg.Config.Child_Address] = boolVal
			b := byte_util.Get_list_index(byte_util.BoolToBytes(bool_list), 0, 2)
			byte_util.Update_List_Slice(&byte_list, int(index)*2, b)
		case cfg.Config.Type == "uint16" && packet.Function == 3:
			val, ok := v.Value.(uint16)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 uint16, 实际 %T", id, v.Value)
			}
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Uint16ToBytes([]uint16{val}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Config.Type == "int16" && packet.Function == 3:
			val, ok := v.Value.(int16)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 int16, 实际 %T", id, v.Value)
			}
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Int16ToBytes([]int16{val}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Config.Type == "uint32" && packet.Function == 3:
			val, ok := v.Value.(uint32)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 uint32, 实际 %T", id, v.Value)
			}
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Uint32ToBytes([]uint32{val}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Config.Type == "int32" && packet.Function == 3:
			val, ok := v.Value.(int32)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 int32, 实际 %T", id, v.Value)
			}
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Int32ToBytes([]int32{val}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Config.Type == "float32" && packet.Function == 3:
			val, ok := v.Value.(float32)
			if !ok {
				return fmt.Errorf("点位 id=%d 值类型断言失败: 期望 float32, 实际 %T", id, v.Value)
			}
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Float32ToBytes([]float32{val}, cfg.Config.Byte_Order),
				0, 2))
		default:
			return fmt.Errorf("不支持的写入类型: type=%s, function=%d, 点位id=%d", cfg.Config.Type, packet.Function, id)
		}
	}
	switch packet.Function {
	case 1:
		a := byte_util.Get_list_index(byte_list, 0, int(packet.Number_Address))
		return (*c.conn).WriteMultipleCoils(packet.SlaveID, packet.Start_Address, packet.Number_Address, a)
	case 3:
		a := byte_util.Get_list_index(byte_list, 0, int(packet.Number_Address*2))
		return (*c.conn).WriteMultipleRegistersBytes(packet.SlaveID, packet.Start_Address, packet.Number_Address, a)
	}
	log.Printf("WARN 驱动 id=%d 写入未匹配到有效功能码: function=%d", c.Drive.Id, packet.Function)
	return fmt.Errorf("未匹配到有效写入功能码: function=%d", packet.Function)
}

// 写入外部映射
func (c *Modbus_Tcp) Write(values []fullConfig.Value_type) (err error) {
	var points []Point_Config_type
	tag_points_map := make(map[uint]fullConfig.Value_type)
	for _, v := range values {
		tag_points_map[v.PointId] = v
		var cfg Point_Config_type
		cfg, err = c.id_points_index(v.PointId)
		if err != nil {
			return err
		}
		points = append(points, cfg)
	}

	var packets []Packet_type
	packets, err = c.packet(points, map[int]bool{3: true, 4: true}) // 3=只写(W) 4=读写(R/W)
	if err != nil {
		return fmt.Errorf("ERROR 组包失败: %v", err)
	}

	for _, packet := range packets {
		err = c.write_packet(packet, tag_points_map)
		if err != nil {
			log.Print(err)
			continue
		}
	}

	return
}
