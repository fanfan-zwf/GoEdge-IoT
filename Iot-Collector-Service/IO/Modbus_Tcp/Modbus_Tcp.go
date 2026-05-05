/*
* 日期: 2025.5.13 PM17:26
* 作者: 范范zwf
* 作用: Connect驱动
 */

package Modbus_Tcp

import (
	"main/Init"
	"sort"

	"bytes"
	"errors"
	"fmt"
	"log"
	"net"

	"sync"
	"time"
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

// 写值
type Posted_value struct {
	Points_Id  uint
	Value_Type string      // 值类型
	Value      interface{} // 值
}

// 驱动更新值
type Get_Value_type struct {
	Points_Id  uint   // 点位id
	Comments   string // 状态
	Value_Type string // 值类型

	Value interface{} // 值
	Time  string      // 时间戳
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
}

type Points_type struct {
	SlaveID       uint8  // 从机地址
	Function      string // Modbus功能码（如3=读保持寄存器）
	Address       uint16 // 寄存器地址
	Type          string // 数据类型（bool/int8/float32等）
	Decimal       uint8  // 小数位数
	Child_Address uint8  // 子地址（可选）
	Byte_Order    string // 字节序（如"ABCD"表示大端）
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
type Mysql_Config_type struct {
	Id     uint   // 驱动id
	Type   string // 驱动类型
	Name   string // 驱动名称
	Config Config_type
}

type Mysql_Points_type struct {
	Id         uint   // 点位id
	Drive_Id   uint   // 驱动id唯一标识符
	Drive_Type string // 驱动类型
	Name       string // 点位名称
	Group      string // 分组
	RW_Cancel  string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type string // 输出类型
	Config     Points_type
}

// 组包
type Packet_type struct {
	SlaveID        uint8  // 设备id
	Function       string // 功能码
	Start_Address  uint16 // 开始地址
	Number_Address uint16 // 地址数量
	PointsId       []uint // 这个包的点位
}

/*******************驱动连接*******************/

type Read_Points_type struct {
	SlaveID  uint8  // 设备id
	Function string // Modbus功能码（如3=读保持寄存器）
	Address  uint16 // 寄存器地址
	Number   uint16 // 寄存器地址
}

// 定义一个结构体
type Connect_struct struct {
	Drive                    Mysql_Config_type   // 通信参数结构体
	Points                   []Mysql_Points_type // 点位结构体
	conn                     net.Conn            // tcp连接实例
	conn_err                 error               //  连接状态
	tcp_sync                 sync.Mutex          // tcp线程锁防止并发
	tcp_again_Connect        sync.Mutex          // 掉线重新锁防止并发
	Data_Packet_Print_enable bool                // 数据包使能
	transaction_ID           uint16              // 事务元标识符
	Read_Points              []Read_Points_type  // 读取结构体
	Packets                  []Packet_type       // 组包格式
	Esc_collection           chan bool

	Tag_Pointsindex_Map map[string]int // tag点位index索引
}

// 定义接口
type Connect_interface interface {
	NewTCPClient(Ip string, Port uint16) error // 初始化连接
	Connect() error                            // 开始连接
	keepAliveConnect() error                   // 掉线重连
	Close() error                              // 关闭连接
	tcp_data(message *[]byte) ([]byte, error)  // 发送tcp数据

	Packet() error                                                        // 组包
	Read(i int) ([]Get_Value_type, error)                                 // 读取具体的包
	Read_Continuous(stopChan *chan bool, Callback func([]Get_Value_type)) // 连续读取

	Write(point_id uint, value any) (exist bool, err error) // 写值

	// 读取  00001至09999是离散输出(线圈)01功能码
	// Start开始地址  Number个数
	Read__Coils_status(Device uint8, Start uint16, Number uint16) ([]bool, error)

	// 读取  10001至19999是离散输入(触点)02功能码
	// Start开始地址  Number个数
	Read__Input_status(Device uint8, Start uint16, Number uint16) ([]bool, error)

	// 读取 30001至39999是输入寄存器(通常是模拟量输入) 04功能码
	// Start开始地址  Number个数
	Read__Input_register(Device uint8, Start uint16, Number uint16) ([]byte, error)

	// 读取  40001至49999是保持寄存器 03功能码
	// Start开始地址  Number个数
	Read__Holding_register(Device uint8, Start uint16, Number uint16) ([]byte, error)

	// 写入单个  00001至09999是离散输出(线圈)
	// Start开始地址  Number个数
	Write_single__Coils_tatus(Device uint8, Start uint16, Value bool) error

	// 写入单个 40001至49999是输入寄存器(通常是模拟量输入)
	// Start开始地址  Number个数
	Write_single__Input_register(Device uint8, Start uint16, Value [2]byte) error

	// 写入多个  00001至09999是离散输出(线圈)
	// Start开始地址  Number个数
	Write_many__Coils_tatus(Device uint8, Start uint16, Number uint16, Value []bool) error

	// 写入多个 40001至49999是输入寄存器(通常是模拟量输入)
	// Start开始地址  Number个数
	Write_many__Input_register(Device uint8, Start uint16, Number uint16, Value []byte) error
}

type Packet_df struct {
	SlaveID  uint8  // 设备id
	Function string // 功能码
}

func (c *Connect_struct) Packet() error {
	// 1️⃣ 初始化 map（必须！否则 panic）
	pointMap := make(map[Packet_df][]PackAddressPackages_Point_type)

	c.Tag_Pointsindex_Map = make(map[string]int)
	// 2️⃣ 遍历点位，按 SlaveID + Function 分组
	for i, point := range c.Points {

		c.Tag_Pointsindex_Map[point.Name] = i // 添加点位和点位配置的索引

		// 构建 key
		key := Packet_df{
			SlaveID:  point.Config.SlaveID,
			Function: point.Config.Function,
		}

		len, exist := Type_byte[point.Config.Type]
		if !exist {
			log.Printf("ERROR modbus_tcp: 无效类型:%s  点位:%s", point.Config.Type, point.Name)
			continue
		}

		// 加入分组
		pointMap[key] = append(pointMap[key], PackAddressPackages_Point_type{
			PointID:   point.Id,
			StartAddr: point.Config.Address,
			DataLen:   len,
		})
	}
	fmt.Printf("\n pointMappointMap  %+v \n", pointMap)
	var Packets []Packet_type
	for key, value := range pointMap {
		packa, err := PackAddressPackages(value, uint16(c.Drive.Config.Packet_max))
		if err != nil {
			log.Printf("ERROR modbus_tcp: 组包错误:%v", err)
			continue
		}

		for _, v := range packa {
			Packets = append(c.Packets, Packet_type{
				SlaveID:        key.SlaveID,
				Function:       key.Function,
				Start_Address:  v.StartAddr,
				Number_Address: v.DataLen,
				PointsId:       v.PointIDs,
			})
		}
	}

	fmt.Printf("Packeaaaaats >>>>>>>> \n%+v \n", Packets)
	c.Packets = Packets

	return nil
}

// 初始化连接
// func (c *Connect_struct) NewTCPClient(Ip string, Port uint16) error {

// 	parsedIP := net.ParseIP(Ip)
// 	if parsedIP == nil {
// 		return errors.New("IP error")
// 	}
// 	if !(strings.Contains(Ip, ".") && parsedIP.To4() != nil) {
// 		return errors.New("IP error")
// 	}
// 	c.Drive.Config.Ip = Ip
// 	c.Drive.Config.Port = Port
// 	return nil
// }

// 开始连接
func (c *Connect_struct) Connect() error {
	c.Esc_collection = make(chan bool)
	// 连接服务器
	Host := fmt.Sprintf("%d", c.Drive.Config.Port)
	address := net.JoinHostPort(c.Drive.Config.Ip, Host)

	for {
		if c.Drive.Config.Connect_timeout == 0 {
			c.Drive.Config.Connect_timeout = 3000
		}
		c.conn, c.conn_err = net.DialTimeout("tcp", address, c.Drive.Config.Connect_timeout)
		if c.conn_err != nil {
			if c.Drive.Config.Retry_timeout == 0 {
				c.Drive.Config.Retry_timeout = 180000
			}
			log.Printf("INFO modbus_tcp: 驱动id:%d,驱动名称:%s,等待:%dms后重连,连接错误:%v ", c.Drive.Id, c.Drive.Name, c.Drive.Config.Retry_timeout, c.conn_err)
			time.Sleep(c.Drive.Config.Retry_timeout) // 等待重连
		} else {
			break
		}
	}

	return c.conn_err
}

// 关闭连接
func (c *Connect_struct) Close() error {
	log.Print("INFO ", c.Drive.Id, "关闭IO")
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// 掉线重新连接
func (c *Connect_struct) keepAliveConnect() error {
	if c.tcp_again_Connect.TryLock() {
		if c.conn != nil {
			c.Close()
			c.conn = nil
		}

		err := c.Connect()

		c.tcp_again_Connect.Unlock()

		return err

	} else {
		return errors.New("等待重新连接")
	}

}

// 发送tcp数据
func (c *Connect_struct) tcp_data(send *[]byte) ([]byte, error) {
	// // 线程锁
	c.tcp_sync.Lock()

	defer func() {
		if !c.tcp_sync.TryLock() {
			c.tcp_sync.Unlock()
		}
	}()
	if c.conn_err != nil {
		return []byte{}, c.conn_err
	}

	if c.conn == nil {
		return []byte{}, errors.New("等待连接")
	}

	c.transaction_ID++
	Start_byte, err := Byte_Convert_uint16_byte(c.transaction_ID, "AB")
	if err != nil {
		return []byte{}, err
	}
	(*send)[0] = Start_byte[0]
	(*send)[1] = Start_byte[1]

	Number_byte, err := Byte_Convert_uint16_byte(uint16(len(*send)-6), "AB")
	if err != nil {
		return []byte{}, err
	}
	(*send)[4] = Number_byte[0]
	(*send)[5] = Number_byte[1]

	// 发送数据
	_, err = c.conn.Write(*send)
	if err != nil {
		go c.keepAliveConnect()
		log.Printf("ERROR %v", err.Error())
		return []byte{}, err
	}

	if c.Data_Packet_Print_enable {
		fmt.Printf("发送:% x\n", (*send))
	}

	// 设置读取超时
	if c.Drive.Config.Response_timeout == 0 {
		c.Drive.Config.Response_timeout = 20
	}
	err = c.conn.SetReadDeadline(time.Now().Add(c.Drive.Config.Response_timeout))
	if err != nil {
		return []byte{}, err
	}

	// 接收响应
	receive := make([]byte, 1024)
	_, err = c.conn.Read(receive)

	if err != nil {
		log.Printf("ERROR %v", err.Error())
		// go c.keepAliveConnect() // 见鬼经常报错
		return []byte{}, err
	}

	// 获取返回长度
	n, err := Byte_Convert_byte_uint16([2]byte{receive[4], receive[5]}, "AB")
	if err != nil {
		return []byte{}, errors.New("长度计算错误")
	}
	n = n + 6
	if c.Data_Packet_Print_enable {
		fmt.Printf("接收:% x\n", receive[:n])
	}

	// 判断事务元标识符是否一致
	if !((*send)[0] == receive[0] && (*send)[1] == receive[1]) {
		return []byte{}, errors.New("事务元标识符错误")
	}
	return receive[:n], err
}

// 读取  00001至09999是离散输出(线圈)
// Start开始地址  Number个数
func (c *Connect_struct) Read__Coils_status(Device uint8, Start uint16, Number uint16) ([]bool, error) {

	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return []bool{}, err
	}
	Number_byte, err := Byte_Convert_uint16_byte(Number, "AB")
	if err != nil {
		return []bool{}, err
	}

	PDU := []byte{
		0x00, 0x00, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x01,                         // 功能码
		Start_byte[0], Start_byte[1], // 开始地址
		Number_byte[0], Number_byte[1], // 长度
	}

	crc := ModbusCRC16(PDU[5:])
	PDU = append(PDU, byte(crc), byte(crc>>8))

	// 读取tcp数据
	r, err := c.tcp_data(&PDU)
	if err != nil {
		return []bool{}, err
	}

	// 判断设备id是否正确
	if r[6] != Device {
		return []bool{}, errors.New("设备id不一致")
	}

	// 异常响应报文
	if r[7] == 0x81 {
		switch r[8] {
		case 0x01:
			return []bool{}, errors.New("从站不支持该功能码")
		case 0x02:
			return []bool{}, errors.New("寄存器地址不存在")
		case 0x03:
			return []bool{}, errors.New("写入的值超出范围")
		case 0x04:
			return []bool{}, errors.New("设备内部错误")
		default:
			return []bool{}, fmt.Errorf("错误码：%x", r[8])
		}
	}
	if r[7] != 0x01 { // 判断功能码是否正确
		return []bool{}, errors.New("功能码错误")
	}

	byte_value := r[9 : 9+int(r[8])]

	var value []bool
	for i := 0; i < len(byte_value); i++ {
		a, err := Byte_Convert_1byte_8bool(byte_value[i])
		if err != nil {
			return []bool{}, err
		}
		value = append(value, a[:]...)
	}
	Number_bool := divideCeil(int(uint(Number)), 7)
	if len(value) < Number_bool {
		return []bool{}, errors.New("组包数量不正确")
	}

	return value[:Number], nil
}

// 读取  10001至19999是离散输入(触点)
// Start开始地址  Number个数
func (c *Connect_struct) Read__Input_status(Device uint8, Start uint16, Number uint16) ([]bool, error) {

	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return []bool{}, err
	}
	Number_byte, err := Byte_Convert_uint16_byte(Number, "AB")
	if err != nil {
		return []bool{}, err
	}

	PDU := []byte{
		0x00, 0x00, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x02,                         // 功能码
		Start_byte[0], Start_byte[1], // 开始地址
		Number_byte[0], Number_byte[1], // 长度
	}

	crc := ModbusCRC16(PDU[5:])
	PDU = append(PDU, byte(crc), byte(crc>>8))

	// 读取tcp数据
	r, err := c.tcp_data(&PDU)
	if err != nil {
		return []bool{}, err
	}

	// 判断设备id是否正确
	if r[6] != Device {
		return []bool{}, errors.New("设备id不一致")
	}

	// 异常响应报文
	if r[7] == 0x81 {
		switch r[8] {
		case 0x01:
			return []bool{}, errors.New("从站不支持该功能码")
		case 0x02:
			return []bool{}, errors.New("寄存器地址不存在")
		case 0x03:
			return []bool{}, errors.New("写入的值超出范围")
		case 0x04:
			return []bool{}, errors.New("设备内部错误")
		default:
			return []bool{}, fmt.Errorf("错误码：%x", r[8])
		}
	}
	if r[7] != 0x02 { // 判断功能码是否正确
		return []bool{}, errors.New("功能码错误")
	}

	byte_value := r[9 : 9+int(r[8])]
	var value []bool
	for i := 0; i < len(byte_value); i++ {
		a, err := Byte_Convert_1byte_8bool(byte_value[i])
		if err != nil {
			return []bool{}, err
		}
		value = append(value, a[:]...)
	}

	Number_bool := divideCeil(int(uint(Number)), 7)
	if len(value) < Number_bool {
		return []bool{}, errors.New("组包数量不正确")
	}
	return value[:Number], nil
}

// 读取  40001至49999是保持寄存器 03功能码
// Start开始地址  Number个数
func (c *Connect_struct) Read__Holding_register(Device uint8, Start uint16, Number uint16) ([]byte, error) {

	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return []byte{}, err
	}
	Number_byte, err := Byte_Convert_uint16_byte(Number, "AB")
	if err != nil {
		return []byte{}, err
	}

	PDU := []byte{
		0x00, 0x00, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x03,                         // 功能码
		Start_byte[0], Start_byte[1], // 开始地址
		Number_byte[0], Number_byte[1], // 长度
	}

	crc := ModbusCRC16(PDU[5:])
	PDU = append(PDU, byte(crc), byte(crc>>8))

	// 读取tcp数据
	r, err := c.tcp_data(&PDU)
	if err != nil {
		return []byte{}, err
	}

	// 判断设备id是否正确
	if r[6] != Device {
		return []byte{}, errors.New("设备id不一致")
	}

	// 异常响应报文
	if r[7] == 0x81 {
		switch r[8] {
		case 0x01:
			return []byte{}, errors.New("从站不支持该功能码")
		case 0x02:
			return []byte{}, errors.New("寄存器地址不存在")
		case 0x03:
			return []byte{}, errors.New("写入的值超出范围")
		case 0x04:
			return []byte{}, errors.New("设备内部错误")
		default:
			return []byte{}, fmt.Errorf("错误码：%x", r[8])
		}
	}
	if r[7] != 0x03 { // 判断功能码是否正确
		return []byte{}, errors.New("功能码错误")
	}

	return r[9:], nil
}

// 读取 30001至39999是输入寄存器(通常是模拟量输入) 04功能码
// Start开始地址  Number个数
func (c *Connect_struct) Read__Input_register(Device uint8, Start uint16, Number uint16) ([]byte, error) {

	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return []byte{}, err
	}
	Number_byte, err := Byte_Convert_uint16_byte(Number, "AB")
	if err != nil {
		return []byte{}, err
	}

	PDU := []byte{
		0x00, 0x00, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x04,                         // 功能码
		Start_byte[0], Start_byte[1], // 开始地址
		Number_byte[0], Number_byte[1], // 长度
	}

	crc := ModbusCRC16(PDU[5:])
	PDU = append(PDU, byte(crc), byte(crc>>8))

	// 读取tcp数据
	r, err := c.tcp_data(&PDU)
	if err != nil {
		return []byte{}, err
	}

	// 判断设备id是否正确
	if r[6] != Device {
		return []byte{}, errors.New("设备id不一致")
	}

	// 异常响应报文
	if r[7] == 0x81 {
		switch r[8] {
		case 0x01:
			return []byte{}, errors.New("从站不支持该功能码")
		case 0x02:
			return []byte{}, errors.New("寄存器地址不存在")
		case 0x03:
			return []byte{}, errors.New("写入的值超出范围")
		case 0x04:
			return []byte{}, errors.New("设备内部错误")
		default:
			return []byte{}, fmt.Errorf("错误码：%x", r[8])
		}
	}
	if r[7] != 0x04 { // 判断功能码是否正确
		return []byte{}, errors.New("功能码错误")
	}

	return r[9:], nil
}

// 写入单个  00001至09999是离散输出(线圈)
// Start开始地址  Number个数
func (c *Connect_struct) Write_single__Coils_tatus(Device uint8, Start uint16, Value bool) error {
	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return err
	}

	send := []byte{
		0x00, 0x01, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x05,                         // 功能码
		Start_byte[0], Start_byte[1], // 线圈地址
		0xff, 0x00, // ​强制值。​只有 0xFF 00表示 ON，0x00 00表示 OFF。其他任何值都是非法的
	}

	if Value {
		send[10] = 0xff
	} else {
		send[10] = 0x00
	}

	// 发送tcp数据
	receive, err := c.tcp_data(&send)
	if err != nil {
		return err
	}

	switch {
	case receive[6] != Device:
		return errors.New("设备id不一致")
	case receive[7] == 0x85 && receive[8] == 0x01:
		return errors.New("从站不支持该功能码")
	case receive[7] == 0x85 && receive[8] == 0x02:
		return errors.New("寄存器地址不存在")
	case receive[7] == 0x85 && receive[8] == 0x03:
		return errors.New("写入的值超出范围")
	case receive[7] == 0x85 && receive[8] == 0x04:
		return errors.New("设备内部错误")
	case receive[7] == 0x05 && bytes.Equal(send, receive):
		return nil
	}

	return fmt.Errorf("错误码：% x", receive[7:])
	// return nil
}

// 写入多个  00001至09999是离散输出(线圈)
// Start开始地址  Number个数
func (c *Connect_struct) Write_many__Coils_tatus(Device uint8, Start uint16, Number uint16, Value []bool) error {
	Value = Value[:Number]

	// 开始地址转化2个字节
	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return err

	}

	// 数组值长度
	Value_Number, err := Byte_Convert_int16_byte(int16(len(Value)), "AB")
	if err != nil {
		return err

	}

	Value_byte := BoolsToBytesLittleEndian(Value)

	Value_byte_Number := len(Value_byte)
	if Value_byte_Number > 120 {
		return errors.New("写值过大")
	}

	send := []byte{
		0x00, 0x01, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x0f,                         // 功能码
		Start_byte[0], Start_byte[1], // 起始地址
		Value_Number[0], Value_Number[1], // 线圈数量
		byte(Value_byte_Number), // 字节数
		// 0xFF, 0x00, // ​强制数据
	}

	send = append(send, Value_byte...)

	// 发送tcp数据
	receive, err := c.tcp_data(&send)
	if err != nil {
		return err
	}

	switch {
	case receive[6] != Device:
		return errors.New("设备id不一致")
	case receive[7] == 0x8F && receive[8] == 0x01:
		return errors.New("从站不支持该功能码")
	case receive[7] == 0x8F && receive[8] == 0x02:
		return errors.New("寄存器地址不存在")
	case receive[7] == 0x8F && receive[8] == 0x03:
		return errors.New("写入的值超出范围")
	case receive[7] == 0x8F && receive[8] == 0x04:
		return errors.New("设备内部错误")
	case receive[7] == 0x0F && receive[8] == send[8] && receive[9] == send[9] && receive[10] == send[10] && receive[11] == send[11]:
		return nil
	}

	return fmt.Errorf("错误码：% x", receive[7:])

}

// 写入单个 40001至49999是输入寄存器(通常是模拟量输入)
// Start开始地址  Number个数
func (c *Connect_struct) Write_single__Input_register(Device uint8, Start uint16, Value [2]byte) error {

	// 开始地址转化2个字节
	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return err

	}

	send := []byte{
		0x00, 0x01, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x06,                         // 功能码
		Start_byte[0], Start_byte[1], // 起始地址
		Value[0], Value[1], // 寄存器值
	}

	// 发送tcp数据
	receive, err := c.tcp_data(&send)
	if err != nil {
		return err
	}

	switch {
	case receive[6] != Device:
		return errors.New("设备id不一致")
	case receive[7] == 0x86 && receive[8] == 0x01:
		return errors.New("从站不支持该功能码")
	case receive[7] == 0x86 && receive[8] == 0x02:
		return errors.New("寄存器地址不存在")
	case receive[7] == 0x86 && receive[8] == 0x03:
		return errors.New("写入的值超出范围")
	case receive[7] == 0x86 && receive[8] == 0x04:
		return errors.New("设备内部错误")
	case receive[7] == 0x06 && bytes.Equal(send, receive):
		return nil
	}

	return fmt.Errorf("错误码：% x", receive[7:])

}

// 写入多个 40001至49999是输入寄存器(通常是模拟量输入)
// Start开始地址  Number个数
func (c *Connect_struct) Write_many__Input_register(Device uint8, Start uint16, Number uint16, Value []byte) error {
	if len(Value)%2 != 0 {
		return errors.New("Value数组必须是2的倍数")
	}

	Value = Value[:Number*2]

	// 开始地址转化2个字节
	Start_byte, err := Byte_Convert_uint16_byte(Start, "AB")
	if err != nil {
		return err

	}

	// 数组值长度
	Value_Number, err := Byte_Convert_uint16_byte(Number, "AB")
	if err != nil {
		return err

	}

	Value_byte_Number := len(Value)
	if Value_byte_Number > 120 {
		return errors.New("写值过大")
	}

	send := []byte{
		0x00, 0x01, // 事务元标识符
		0x00, 0x00, // 协议标识符
		0x00, 0x06, // 长度
		Device,                       // 设备id
		0x10,                         // 功能码
		Start_byte[0], Start_byte[1], // 起始地址
		Value_Number[0], Value_Number[1], // 线圈数量
		byte(Value_byte_Number), // 字节数
		// 0xFF, 0x00, // ​强制数据
	}

	send = append(send, Value...)

	// 发送tcp数据
	receive, err := c.tcp_data(&send)
	if err != nil {
		return err
	}

	switch {
	case receive[6] != Device:
		return errors.New("设备id不一致")
	case receive[7] == 0x90 && receive[8] == 0x01:
		return errors.New("从站不支持该功能码")
	case receive[7] == 0x90 && receive[8] == 0x02:
		return errors.New("寄存器地址不存在")
	case receive[7] == 0x90 && receive[8] == 0x03:
		return errors.New("写入的值超出范围")
	case receive[7] == 0x90 && receive[8] == 0x04:
		return errors.New("设备内部错误")
	case receive[7] == 0x10 && receive[8] == send[8] && receive[9] == send[9] && receive[10] == send[10] && receive[11] == send[11]:
		return nil
	}

	return fmt.Errorf("错误码：% x", receive[7:])

}

type IO_Collection_Value_type struct {
	Points_Id  uint   // 点位id
	Msg        string // 状态
	Value_Type string // 值类型

	Value any // 值
	Time  string
}

// 读取一个包
func (c *Connect_struct) Read(i int) []IO_Collection_Value_type {
	if i < 0 {
		return []IO_Collection_Value_type{}
	}
	if i >= len(c.Packets) {
		return []IO_Collection_Value_type{}
	}
	Packet := c.Packets[i]
	switch Packet.Function {
	case "01":
		v, err := c.Read__Coils_status(Packet.SlaveID, Packet.Start_Address-1, Packet.Number_Address)
		if err != nil {
			value_array := c.read_status_err(err.Error(), Packet.PointsId)
			return value_array
		} else {
			value_array := c.read_status_ok(v, Packet)
			return value_array
		}
	case "02":
		v, err := c.Read__Input_status(Packet.SlaveID, Packet.Start_Address-1, Packet.Number_Address)
		if err != nil {
			value_array := c.read_status_err(err.Error(), Packet.PointsId)
			return value_array
		} else {
			value_array := c.read_status_ok(v, Packet)
			return value_array
		}
	case "03":
		v, err := c.Read__Holding_register(Packet.SlaveID, Packet.Start_Address-1, Packet.Number_Address)
		if err != nil {
			value_array := c.read_register_err(err.Error(), Packet.PointsId)
			return value_array
		} else {
			value_array := c.read_register_ok(v, Packet)
			return value_array
		}
	case "04":
		v, err := c.Read__Input_register(Packet.SlaveID, Packet.Start_Address-1, Packet.Number_Address)
		if err != nil {
			value_array := c.read_register_err(err.Error(), Packet.PointsId)
			return value_array
		} else {
			value_array := c.read_register_ok(v, Packet)
			return value_array
		}
	default:

	}

	return []IO_Collection_Value_type{}

}

func (c *Connect_struct) Read_Continuous(Callback func([]IO_Collection_Value_type)) error {
	log.Printf("INFO modbus_tcp 开始轮询读取: 驱动id:%d,驱动名称:%s", c.Drive.Id, c.Drive.Name)
	var (
		i           int
		Packets_len int
	)

	defer func() {
		var value_collection []IO_Collection_Value_type
		for _, Point := range c.Points {
			if !(Point.RW_Cancel == "W/R" || Point.RW_Cancel == "R") {
				continue
			}
			value_collection = append(value_collection, IO_Collection_Value_type{
				Points_Id:  Point.Id,         // 点位id
				Msg:        "轮询读取结束",         // 状态
				Value_Type: Point.Value_Type, // 值类型
				Time:       time.Now().Format(Init.RFC_FAN),
			})
		}
		Callback(value_collection)
	}()

	for {
		select {
		case <-c.Esc_collection:
			return nil
		default:
			fmt.Printf("\n 论序开始:%d\n", i)
			Get_Value_array := c.Read(i)

			Callback(Get_Value_array)

			time.Sleep(c.Drive.Config.Delay_between_polls)

			i++

			if i >= Packets_len {
				Packets_len = len(c.Packets)
				i = 0
				continue
			}

		}

	}

}

/*
************************写值************************
 */

func (c *Connect_struct) Write(point_id uint, value any) (exist bool, err error) {
	point_config, err := c.query_points_config(point_id)
	if err != nil {
		return false, nil
	}

	if point_config.Id != point_id {
		return false, nil
	}

	if point_config.Drive_Type != "modbus_tcp" {
		log.Printf("ERROR modbus_tcp写值错误:点位设备类型, 点位id%d,驱动%s", point_id, point_config.Drive_Type)
		return true, fmt.Errorf("ERROR modbus_tcp写值错误:点位设备类型, 点位id%d,驱动%s", point_id, point_config.Drive_Type)
	}

	if !(point_config.RW_Cancel == "W/R" || point_config.RW_Cancel == "W") {
		log.Printf("ERROR modbus_tcp写值错误:禁止写, 点位id%d,读写模式%s", point_id, point_config.RW_Cancel)
		return true, fmt.Errorf("ERROR modbus_tcp写值错误:禁止写, 点位id%d,读写模式%s", point_id, point_config.RW_Cancel)
	}

	// 写01功能码值
	if point_config.Config.Function == "01" {
		return true, c.Write_register_Coils_tatus(point_config, value)
	}

	// 写03功能码
	if point_config.Config.Function == "03" {
		return true, c.Write_register_Input_register(point_config, value)
	}

	log.Printf("ERROR modbus_tcp写值错误:未执行, 点位id%d,功能码%s", point_id, point_config.Config.Function)
	return true, fmt.Errorf("功能码错误")

}

// 你定义的结构体（完全保留）
type PackAddressPackages_Point_type struct {
	PointID   uint   // 点位id 唯一的 + 不能为0 + 不能重复
	StartAddr uint16 // 点位开始值
	DataLen   uint16 // 点位类型长度
	EndAddr   uint16 // 内部计算用
}

// 组包结果结构
type PackageResult struct {
	StartAddr uint16
	DataLen   uint16
	PointIDs  []uint // 改成 uint 匹配你的结构
}

// PackAddressPackages 核心组包函数（带去重 + PointID 不能为0）
func PackAddressPackages(addrList []PackAddressPackages_Point_type, maxPackageLen uint16) ([]PackageResult, error) {

	// 内部 max 函数
	max := func(a, b uint16) uint16 {
		if a > b {
			return a
		}
		return b
	}

	// ======================
	//  强制校验：PointID 不能为0 + 不能重复
	// ======================
	pointIDMap := make(map[uint]bool)
	var validPoints []PackAddressPackages_Point_type

	for _, p := range addrList {
		// 1. 不能为 0
		if p.PointID == 0 {
			return nil, fmt.Errorf("错误：点位ID不能为0")
		}
		// 2. 不能重复
		if pointIDMap[p.PointID] {
			return nil, fmt.Errorf("错误：点位ID重复 → %d", p.PointID)
		}
		// 3. 数据长度必须 >0
		if p.DataLen <= 0 {
			return nil, fmt.Errorf("错误：点位ID=%d 数据长度必须>0", p.PointID)
		}

		// 标记已存在
		pointIDMap[p.PointID] = true

		// 计算结束地址
		p.EndAddr = p.StartAddr + p.DataLen - 1
		validPoints = append(validPoints, p)
	}

	// 按地址排序
	sort.Slice(validPoints, func(i, j int) bool {
		return validPoints[i].StartAddr < validPoints[j].StartAddr
	})

	// 组包逻辑
	var packages []PackageResult
	for _, point := range validPoints {
		currStart := point.StartAddr
		currEnd := point.EndAddr

		if len(packages) > 0 {
			lastIdx := len(packages) - 1
			lastPkg := &packages[lastIdx]
			lastStart := lastPkg.StartAddr
			lastEnd := lastStart + lastPkg.DataLen - 1

			if currStart <= lastEnd {
				mergedEnd := max(lastEnd, currEnd)
				mergedLen := mergedEnd - lastStart + 1

				if maxPackageLen == 0 || mergedLen <= maxPackageLen {
					lastPkg.DataLen = mergedLen
					lastPkg.PointIDs = append(lastPkg.PointIDs, point.PointID)
					continue
				}
			}
		}

		// 新建包
		packages = append(packages, PackageResult{
			StartAddr: currStart,
			DataLen:   point.DataLen,
			PointIDs:  []uint{point.PointID},
		})
	}

	return packages, nil
}
