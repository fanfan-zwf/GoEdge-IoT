/*
* 日期: 2025.5.13 PM17:26
* 作者: 范范zwf
* 作用: Connect驱动
 */

package Modbus_Tcp

import (
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
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
	Ip                  string        // IP地址
	Port                uint16        // 端口（可选，默认502）
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

// mysql存储结构体
type Drive_Config_type struct {
	Id     uint   // 驱动id
	Type   string // 驱动类型
	Name   string // 驱动名称
	Config Config_type
}

type Points_Config_type struct {
	Tag        string // 点位标识符
	RW_Cancel  string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type string // 输出类型
	Config     Points_type
}

// 组包
type Packet_type struct {
	SlaveID        uint8    // 设备id
	Function       uint8    // 功能码
	Start_Address  uint16   // 开始地址
	Number_Address uint16   // 地址数量
	Tags           []string // 这个包的点位
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
	fullConfig.BaseDriver                      // 驱动全配置（驱动配置 + 该驱动下的所有点位配置）
	Drive                 Drive_Config_type    // 通信参数结构体
	Points                []Points_Config_type // 点位结构体

	conn          *modbus.Client // tcp连接实例
	conn_err      error          // 连接状态
	first_connect bool           // 首次连接

	read_points    []Read_Points_type // 读取结构体
	packets        []Packet_type      // 组包格式
	Esc_collection chan bool

	Tag_Pointsindex_Map map[string]int // tag点位index索引

	Read_External_Mappings func([]fullConfig.Value_type) error
	Write_value_mu         sync.Mutex
}

// 定义接口
type Connect_interface interface {
	New() error     // 初始化
	Connect() error // 开始连接
	Close() error   // 停止连接
}

func (m *Modbus_Tcp) LoadConfig(cfg fullConfig.FullConfig_type) error {
	m.BaseDriver.LoadConfig(cfg)

	// 解析驱动配置字符串格式: IP;Port;RetryTimeout;ConnectTimeout;ResponseTimeout;DelayBetweenPolls;PacketMax
	var err error
	m.Drive.Config, err = Drive_Config_Switch(cfg.Drive.Config)
	if err != nil {
		return fmt.Errorf("ERROR 解析驱动配置失败: %w", err)
	}

	// 设置其他字段
	m.Drive.Id = cfg.Drive.Id
	m.Drive.Type = cfg.Drive.Type
	m.Drive.Name = cfg.Drive.Name

	return nil
}

type Packet_df struct {
	SlaveID  uint8 // 设备id
	Function uint8 // 功能码
}

func (c *Modbus_Tcp) New() (err error) {
	c.Drive.Config, err = Drive_Config_Switch(c.Config.Drive.Config)
	if err != nil {
		return err
	}

	c.Tag_Pointsindex_Map = make(map[string]int)

	var points []Points_Config_type
	for i, v := range c.Config.Points {
		point, err := Point_Config_Switch(v.Config)
		if err != nil {
			return fmt.Errorf("ERROR 解析点位配置失败: %v, 配置字符串: %s", err, v.Config)
		}
		points = append(points, Points_Config_type{
			Config:     point,
			RW_Cancel:  v.RW_Cancel,
			Tag:        v.Tag,
			Value_Type: v.Value_Type,
		})
		c.Tag_Pointsindex_Map[v.Tag] = i // 建立tag到index的映射

	}
	c.Points = points

	c.packets, err = c.packet(c.Points, map[string]bool{"R": true, "R/W": true})
	if err != nil {
		return fmt.Errorf("ERROR 组包失败: %v", err)
	}

	return nil
}

func (c *Modbus_Tcp) tag_points_index(tag string) (p Points_Config_type, err error) {
	index, exists := c.Tag_Pointsindex_Map[tag]
	if !exists {
		err = fmt.Errorf("ERROR 点位不存在:  c.Tag_Pointsindex_Map:%+ v   tag %s", c.Tag_Pointsindex_Map, tag)
		return
	}

	if index < 0 || index >= len(c.Points) {
		err = fmt.Errorf("ERROR 点位下标越界, index: %d, 切片长度: %d", index, len(c.Points))
		return
	}

	p = c.Points[index]
	return
}
func (c *Modbus_Tcp) packet(Points []Points_Config_type, RW_Cancel map[string]bool) (Packets []Packet_type, err error) {
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
			log.Printf("ERROR modbus_tcp: 无效类型:%s  点位:%s", point.Config.Type, point.Tag)
			continue
		}

		// 加入分组
		pointMap[key] = append(pointMap[key], PackAddressPackages_Point_type{
			Tag:       point.Tag,
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
				Tags:           v.Tags,
			})
		}
	}

	return
}

// 开始连接外部映射
func (c *Modbus_Tcp) Connect() error {
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
	p := modbus.NewTCPClientProvider(
		fmt.Sprintf("%s:%d", c.Drive.Config.Ip, c.Drive.Config.Port),
		modbus.WithTCPTimeout(c.Drive.Config.Connect_timeout),
	)

	client := modbus.NewClient(p)

	c.conn = &client

	c.conn_err = client.Connect()
	if c.conn_err != nil {
		c.Error_External_Mappings(c.conn_err.Error())
		return c.conn_err
	}

	log.Printf("modbus_tcp 连接状态: %v", c.conn_err)
	return nil
}

// 关闭连接
func (c *Modbus_Tcp) Close() error {
	(*c.conn).Close()
	c.Error_External_Mappings("驱动连接已关闭")

	return nil
}

func (c *Modbus_Tcp) Error_External_Mappings(msg string) error {
	var read_list []fullConfig.Value_type
	for _, point := range c.Points {
		read_list = append(read_list, fullConfig.Value_type{
			Tag:  point.Tag,        // 点位名称
			Type: point.Value_Type, // 输出类型
			Msg:  msg,              // 状态信息
			Time: time.Now(),       // 读取时间
		})
	}

	// 外部映射
	if c.Read_External_Mappings != nil {
		c.Read_External_Mappings(read_list)
	}

	return nil
}

func (c *Modbus_Tcp) Error_External_Mappings_list(tags []string, msg string) (err error) {
	var read_list []fullConfig.Value_type
	for _, tag := range tags {
		cfg, err := c.tag_points_index(tag)
		if err != nil {
			log.Printf("ERROR modbus_tcp: %v", err)
			continue
		}
		read_list = append(read_list, fullConfig.Value_type{
			Tag:   tag,            // 点位名称
			Type:  cfg.Value_Type, // 输出类型
			Msg:   msg,            // 状态信息
			Time:  time.Now(),     // 读取时间
			Value: nil,
		})
	}
	// 外部映射
	if c.Read_External_Mappings != nil {
		c.Read_External_Mappings(read_list)
	}

	return nil
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
	for _, tag := range packet.Tags {
		var read fullConfig.Value_type
		cfg, err := c.tag_points_index(tag)
		if err != nil {
			log.Printf("ERROR modbus_tcp: %v", err)
			continue
		}

		read.Time = now
		read.Tag = tag
		read.Type = cfg.Value_Type
		read.Msg = "ok"

		byte_index := int(cfg.Config.Address - packet.Start_Address)
		switch {
		case cfg.Config.Type == "bool" && (packet.Function == 1 || packet.Function == 2):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToBool([]byte{byte_util.Get_list_index(results, byte_index/8, 1)[0]}),
				byte_index%8, 1)[0]
		case cfg.Config.Type == "bool" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToBool([]byte{byte_util.Get_list_index(results, byte_index/16, 1)[0]}),
				int(cfg.Config.Child_Address), 1)[0]
		case cfg.Config.Type == "uint16" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToUint16(
					byte_util.Get_list_index(results, byte_index*2, 2),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "int16" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToInt16(
					byte_util.Get_list_index(results, byte_index*2, 2),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "uint32" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToUint32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "int32" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToInt32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		case cfg.Config.Type == "float32" && (packet.Function == 3 || packet.Function == 4):
			read.Value = byte_util.Get_list_index(
				byte_util.BytesToFloat32(
					byte_util.Get_list_index(results, byte_index*2, 4),
					cfg.Config.Byte_Order,
				),
				0, 1,
			)[0]
		default:
			read.Msg = fmt.Sprintf("ERROR tag: %s, 配置类型: %s", tag, cfg.Value_Type)
		}
		read_list = append(read_list, read)

	}
	return read_list, nil
}

func (c *Modbus_Tcp) polling() {
	var i int
	for {
		if c.conn_err == fmt.Errorf("关闭连接") {
			log.Printf("ERROR 驱动:%s 连接未建立", c.Drive.Name)
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
			c.Error_External_Mappings_list(packet.Tags, "Unknown function code")
		}

		if err != nil {
			log.Printf("ERROR 设备id:%d 读取错误:%v", c.Drive.Id, err)
			c.Error_External_Mappings_list(packet.Tags, err.Error())
			continue
		}

		read_list, err := c.analysis(packet, byte_list)
		if err != nil {
			log.Printf("ERROR 设备id:%d 分析错误:%v", c.Drive.Id, err)
			c.Error_External_Mappings_list(packet.Tags, err.Error())
			continue
		}

		// 外部映射
		if c.Read_External_Mappings != nil {
			c.Read_External_Mappings(read_list)
		}

	}

}

// 写入组包
func (c *Modbus_Tcp) write_packet(packet Packet_type, tag_points_map map[string]fullConfig.Value_type) error {

	bool_value_address := make(map[uint16]bool) // 线圈地址与值的映射

	now := time.Now()
	var byte_list []byte
	for _, tag := range packet.Tags {
		var cfg Points_Config_type
		cfg, err := c.tag_points_index(tag)
		if err != nil {
			err = fmt.Errorf("设备id:%d %v", c.Drive.Id, err)
			log.Print(err)
			return err
		}
		v, exists := tag_points_map[tag]
		if !exists {
			err = fmt.Errorf("ERROR modbus_tcp: 写入值不存在, tag: %s", tag)
			log.Print(err)
			return err
		}

		// 时间确认
		if !v.Time.IsZero() {
			duration := now.Sub(v.Time)
			if duration >= (5*time.Second) || duration <= (5*time.Second) {
				err = fmt.Errorf("ERROR modbus_tcp: 写入值时间间隔过长, tag: %s, 时间间隔: %s", tag, duration)
				log.Print(err)
				return err

			}
		}

		// 类型确认
		if v.Type != cfg.Value_Type {
			err = fmt.Errorf("ERROR modbus_tcp: 配置类型与写入值类型不匹配, tag: %s, 配置类型: %s, 值类型: %s", tag, cfg.Value_Type, v.Type)
			log.Print(err)
			return err
		}

		index := cfg.Config.Address - packet.Start_Address // 计算相对地址索引
		if !byte_util.Is_Type_Match(v.Value, cfg.Value_Type) {
			err = fmt.Errorf("ERROR modbus_tcp: 写入值类型不匹配, tag: %s, 配置类型: %s, 值类型: %T", tag, cfg.Value_Type, v.Value)
			log.Print(err)
			return err
		}

		if !byte_util.Is_Type_Match(v.Value, cfg.Config.Type) {
			err = fmt.Errorf("ERROR modbus_tcp: 写入值类型不匹配, tag: %s, 配置类型: %s, 值类型: %T", tag, cfg.Value_Type, v.Value)
			log.Print(err)
			return err
		}

		switch {
		case cfg.Value_Type == "bool" && packet.Function == 1:
			a := byte_util.Get_list_index(byte_list, int(index)/8, 1)
			b := byte_util.BytesToBool(a)
			b[index] = v.Value.(bool)
			rb := byte_util.BoolToBytes(b)
			byte_util.Update_List_Slice(&byte_list, int(index), rb)
		case cfg.Value_Type == "bool" && packet.Function == 3:
			if !bool_value_address[cfg.Config.Address] {
				byte_list, err = (*c.conn).ReadHoldingRegistersBytes(cfg.Config.SlaveID, cfg.Config.Address, 1)
				if err != nil {
					log.Print(err)
					return err
				}
				byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(byte_list, 0, 2))
			}
			a := byte_util.Get_list_index(byte_list, int(index)*2, 2)
			bool_list := byte_util.Get_list_index(byte_util.BytesToBool(a), 0, 16)
			if cfg.Config.Child_Address > 15 {
				err = fmt.Errorf("ERROR modbus_tcp: 子地址超出范围, tag: %s", tag)
				return err
			}
			bool_list[cfg.Config.Child_Address] = v.Value.(bool)
			b := byte_util.Get_list_index(byte_util.BoolToBytes(bool_list), 0, 2)
			byte_util.Update_List_Slice(&byte_list, int(index)*2, b)
		case cfg.Value_Type == "uint16" && packet.Function == 3:
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Uint16ToBytes([]uint16{uint16(v.Value.(uint16))}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Value_Type == "int16" && packet.Function == 3:
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Int16ToBytes([]int16{int16(v.Value.(int16))}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Value_Type == "uint32" && packet.Function == 3:
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Uint32ToBytes([]uint32{uint32(v.Value.(uint32))}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Value_Type == "int32" && packet.Function == 3:
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Int32ToBytes([]int32{int32(v.Value.(int32))}, cfg.Config.Byte_Order),
				0, 2))
		case cfg.Value_Type == "float32" && packet.Function == 3:
			byte_util.Update_List_Slice(&byte_list, int(index)*2, byte_util.Get_list_index(
				byte_util.Float32ToBytes([]float32{v.Value.(float32)}, cfg.Config.Byte_Order),
				0, 2))
		default:
			err = fmt.Errorf("ERROR modbus_tcp: 不支持的类型: %s, tag: %s", cfg.Value_Type, tag)
			log.Print(err)
			return err
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
	log.Print("ERROR 未执行")
	return fmt.Errorf("ERROR 未执行")
}

// 写入外部映射
func (c *Modbus_Tcp) Write(values []fullConfig.Value_type) (err error) {
	var points []Points_Config_type
	tag_points_map := make(map[string]fullConfig.Value_type)
	for _, v := range values {
		tag_points_map[v.Tag] = v
		var cfg Points_Config_type
		cfg, err = c.tag_points_index(v.Tag)
		if err != nil {
			return err
		}
		points = append(points, cfg)
	}

	var packets []Packet_type
	packets, err = c.packet(points, map[string]bool{"R/W": true, "W": true})
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
