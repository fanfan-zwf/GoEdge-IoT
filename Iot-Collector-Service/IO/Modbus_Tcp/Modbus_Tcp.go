/*
* 日期: 2025.5.13 PM17:26
* 作者: 范范zwf
* 作用: Connect驱动
 */

package Modbus_Tcp

import (
	"main/IO/byte_convert"

	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// 类型字节数量输出
var Type_byte = map[string]int{
	"bool":    1,
	"int16":   1,
	"uint16":  1,
	"int32":   2,
	"uint32":  2,
	"float32": 2,
}

type Packets_type struct {
	SlaveID  uint8  // 从机地址
	Function string // Modbus功能码（如3=读保持寄存器）
}

// 写值
type Posted_value struct {
	Points_Id uint

	Value_Type string      // 值类型
	Value      interface{} // 值
}

// 驱动更新值
type Get_Value_type struct {
	PC_Id      string // 设备id
	Points_Id  uint   // 点位id
	Comments   string // 状态
	Value_Type string // 值类型

	Value interface{} // 值
	Time  string      // 时间戳
}

/*******************驱动配置*******************/

type Config_type struct {
	Ip                  string `json:"Ip"`                  // IP地址
	Port                uint16 `json:"Port"`                // 端口（可选，默认502）
	Retry_timeout       uint   `json:"Retry_timeout"`       // 重试间隔（可选，默认3000）
	Connect_timeout     uint   `json:"Connect_timeout"`     // 连接超时（可选，默认3000）
	Response_timeout    uint   `json:"Response_timeout"`    // 响应超时（可选，默认180000)
	Delay_between_polls uint   `json:"Delay_between_polls"` // 轮询时间（可选，默认1000）
	Packet_max          uint8  `json:"Packet_max"`          // 组包字节个数
}

type Points_type struct {
	SlaveID       uint8  `json:"SlaveID"`       // 从机地址
	Function      string `json:"Function"`      // Modbus功能码（如3=读保持寄存器）
	Address       uint16 `json:"Address"`       // 寄存器地址
	Type          string `json:"Type"`          // 数据类型（bool/int8/float32等）
	Decimal       uint8  `json:"Decimal"`       // 小数位数
	Child_Address uint8  `json:"Child_Address"` // 子地址（可选）
	Byte_Order    string `json:"Byte_Order"`    // 字节序（如"ABCD"表示大端）
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
	Id     uint
	Config Config_type
}

type Mysql_Points_type struct {
	Tag    string
	Config Points_type
}

// 组包
type Packet_type struct {
	SlaveID        uint8    // 设备id
	Function       string   // 功能码
	Start_Address  uint16   // 开始地址
	Number_Address uint16   // 地址数量
	PointsId       []string // 这个包的点位
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
	Drive                 Mysql_Config_type   // 通信参数结构体
	Points                []Mysql_Points_type // 点位结构体
	PointsConfig_PointId  map[uint]int
	PointsConfig_PointTag map[string]int
	Packets               []Packet_type // 组包格式

	conn                     net.Conn   // tcp连接实例
	conn_err                 error      //  连接状态
	tcp_sync                 sync.Mutex // tcp线程锁防止并发
	tcp_again_Connect        sync.Mutex // 掉线重新锁防止并发
	Data_Packet_Print_enable bool       // 数据包使能
	transaction_ID           uint16     // 事务元标识符

	Esc_collection chan bool

	Receive_Response int8 // 接收相应超时次数
}

// 定义接口
type Connect_interface interface {
	NewTCPClient(Ip string, Port uint16) error // 初始化连接
	Connect() error                            // 开始连接
	keepAliveConnect() error                   // 掉线重连
	Close() error                              // 关闭连接
	tcp_data(message *[]byte) ([]byte, error)  // 发送tcp数据

	Packet() error // 组包

	// CheckTransactionID 校验Modbus响应的事务标识符是否匹配
	CheckTransactionID(send []byte, receive []byte) error

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

func (c *Connect_struct) PointsConfig_Tag(Tags string) (Point Mysql_Points_type, err error) {
	index, ok := c.PointsConfig_PointTag[Tags]
	if !ok {
		err = fmt.Errorf("ERROR 不存在的点位标识")
		return
	}
	if index < 0 || index >= len(c.Points) {
		err = fmt.Errorf("点位下标越界, index: %d, 切片长度: %d", index, len(c.Points))
		return
	}
	Point = c.Points[index]
	return
}

// 组包
func (c *Connect_struct) Packet() error {
	var err error

	return err
}

// 初始化连接
func (c *Connect_struct) NewTCPClient(Ip string, Port uint16) error {

	parsedIP := net.ParseIP(Ip)
	if parsedIP == nil {
		return errors.New("IP error")
	}
	if !(strings.Contains(Ip, ".") && parsedIP.To4() != nil) {
		return errors.New("IP error")
	}
	c.Drive.Config.Ip = Ip
	c.Drive.Config.Port = Port
	return nil
}

// 开始连接
func (c *Connect_struct) Connect() error {
	c.Esc_collection = make(chan bool)
	c.Receive_Response = 0

	// 连接服务器
	Host := fmt.Sprintf("%d", c.Drive.Config.Port)
	address := net.JoinHostPort(c.Drive.Config.Ip, Host)

	if c.Drive.Config.Connect_timeout == 0 {
		c.Drive.Config.Connect_timeout = 3000
	}

	// 单次尝试连接，不再内部无限循环，由调用者或keepAlive决定重试
	conn, err := net.DialTimeout("tcp", address, time.Duration(c.Drive.Config.Connect_timeout)*time.Millisecond)
	if err != nil {
		log.Printf("WARN modbus_tcp: 驱动id:%d, 连接失败: %v", c.Drive.Id, err)
		c.conn_err = err
		return err
	}

	c.conn = conn
	c.conn_err = nil
	log.Printf("INFO modbus_tcp: 驱动id:%d, 连接成功", c.Drive.Id)
	return nil
}

// 关闭连接
func (c *Connect_struct) Close() error {
	log.Print("INFO ", c.Drive.Id, "关闭IO")
	c.conn_err = fmt.Errorf("关闭IO")
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// 掉线重新连接
func (c *Connect_struct) keepAliveConnect() error {
	// 尝试获取锁，如果获取失败说明其他协程正在重连，直接返回错误避免并发重连
	if !c.tcp_again_Connect.TryLock() {
		// 等待一小段时间后再次检查连接状态，如果其他协程重连成功，则直接返回
		time.Sleep(100 * time.Millisecond)
		if c.conn != nil && c.conn_err == nil {
			return nil
		}
		return errors.New("等待重新连接")
	}
	defer c.tcp_again_Connect.Unlock()

	// 双重检查：如果在等待锁的过程中连接已经恢复，则无需重连
	if c.conn != nil && c.conn_err == nil {
		// 简单探测连接是否存活，可选
		return nil
	}

	log.Printf("INFO modbus_tcp: 驱动id:%d, 开始执行掉线重连...", c.Drive.Id)

	// 安全关闭旧连接
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	// 重置连接错误状态，以便 Connect 尝试新连接
	c.conn_err = nil

	// 执行重连
	err := c.Connect()
	if err != nil {
		log.Printf("ERROR modbus_tcp: 驱动id:%d, 重连失败: %v", c.Drive.Id, err)
		// 设置一个标记错误，表明当前处于断开状态
		c.conn_err = errors.New("重连失败")
		return err
	}

	log.Printf("INFO modbus_tcp: 驱动id:%d, 重连成功", c.Drive.Id)
	return nil
}

// CheckTransactionID 校验Modbus响应的事务标识符是否匹配
func (c *Connect_struct) CheckTransactionID(send []byte, receive []byte) error {
	// 1. 先校验切片长度，避免索引越界
	if len(send) < 2 {
		return errors.New("ERROR 发送报文长度不足,无法获取事务标识符")
	}
	if len(receive) < 2 {
		return errors.New("ERROR 接收报文长度不足,无法获取事务标识符")
	}

	// 2. 正确解析事务标识符（Modbus TCP用大端序）
	sendTID := binary.BigEndian.Uint16(send[:2])    // 发送的事务ID
	recvTID := binary.BigEndian.Uint16(receive[:2]) // 接收的事务ID

	// 3. 对比事务ID
	if sendTID != recvTID {
		return fmt.Errorf("事务元标识符错误：发送%d, 接收%d, 客户端记录%d",
			sendTID, recvTID, c.transaction_ID)
	}

	return nil
}

// 发送tcp数据
func (c *Connect_struct) tcp_data(send *[]byte) ([]byte, error) {
	// 检查连接状态，如果已知断开，先尝试重连
	if c.conn_err != nil || c.conn == nil {
		if err := c.keepAliveConnect(); err != nil {
			return []byte{}, err
		}
	}

	// 线程锁
	c.tcp_sync.Lock()
	defer c.tcp_sync.Unlock()

	// 再次检查连接状态，防止在等待锁期间连接断开
	if c.conn == nil {
		return []byte{}, errors.New("连接未建立")
	}

	c.transaction_ID++
	if c.transaction_ID == 0 || c.transaction_ID > 100 {
		c.transaction_ID = 2
	}

	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{c.transaction_ID}, byte_convert.AB)
	(*send)[0] = Start_byte[0]
	(*send)[1] = Start_byte[1]

	Number_byte := byte_convert.Convert_uint16_uint8([]uint16{uint16((len(*send) - 6))}, byte_convert.AB)
	(*send)[4] = Number_byte[0]
	(*send)[5] = Number_byte[1]

	// 发送数据
	_, err := c.conn.Write(*send)
	if err != nil {
		log.Printf("ERROR modbus_tcp: 发送数据失败: %v", err)
		// 标记连接错误，触发下次重连
		c.conn_err = err
		// 尝试异步重连或在此处处理，通常建议返回错误由上层处理或触发重连
		// 这里为了保持原有逻辑风格，尝试立即重连可能会因为锁的问题复杂化，
		// 故仅标记错误并关闭连接，让下一次调用 tcp_data 时触发 keepAliveConnect
		_ = c.conn.Close()
		c.conn = nil
		return []byte{}, err
	}

	if c.Data_Packet_Print_enable {
		fmt.Printf("发送:% x\n", (*send))
	}

	// 设置读取超时
	if c.Drive.Config.Response_timeout == 0 {
		c.Drive.Config.Response_timeout = 20
	}
	err = c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.Drive.Config.Response_timeout) * time.Millisecond))
	if err != nil {
		c.conn_err = err
		return []byte{}, err
	}

	// 新代码：读取全部响应数据
	// 步骤1：先读取6字节报文头（Modbus TCP固定头）
	header := make([]byte, 6)
	nHeader, err := io.ReadFull(c.conn, header)
	if err != nil {
		// 处理读取头的错误（比如超时、连接断开）
		c.Receive_Response += 1
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("WARN modbus_tcp: 读取响应头超时：%v", err)
		} else {
			log.Printf("ERROR modbus_tcp: 读取响应头失败：%v", err)
			// 非超时错误通常意味着连接断开
			c.conn_err = err
			_ = c.conn.Close()
			c.conn = nil
		}

		// 连续超时或错误次数过多，主动触发重连
		// 注意：此处不再异步调用，而是依靠上层或下一次调用时的检查
		// 如果业务强依赖即时重连，可在此处同步调用，但需注意死锁风险（当前已有tcp_sync锁）
		// 由于 keepAliveConnect 需要 tcp_again_Connect 锁，而当前持有 tcp_sync (虽然目前看没有)，则会死锁。
		// 目前 keepAliveConnect 只获取 tcp_again_Connect，所以同步调用是安全的，但会阻塞当前读取。
		// 鉴于 Modbus 是请求-响应模式，阻塞当前请求直到重连完成或失败是合理的。
		if c.Receive_Response > 3 {
			log.Printf("WARN modbus_tcp: 连续响应异常，触发重连")
			// 释放 tcp_sync 锁以避免潜在的死锁或长时间占用
			c.tcp_sync.Unlock()
			reconnectErr := c.keepAliveConnect()
			c.tcp_sync.Lock() // 重新加锁以符合 defer Unlock
			if reconnectErr != nil {
				return []byte{}, reconnectErr
			}
			// 重连成功后，可能需要重新发送请求，或者由上层重试
			// 这里简单返回错误，让上层决定是否需要重试
			return []byte{}, errors.New("触发重连，请重试请求")

		}
		return []byte{}, err
	}
	if nHeader != 6 {
		return []byte{}, errors.New("响应头长度不足6字节，读取不完整")
	}

	// 步骤2：解析长度字段（第5-6字节，大端序）
	// 注意：这里复用你原有的字节转换函数
	// n, err := byte_convert.Byte_Convert_byte_uint16([2]byte{header[4], header[5]}, "AB")
	n := byte_convert.Convert_uint8_uint16([]byte{header[4], header[5]}, byte_convert.AB)[0]

	// n 是「单元ID+数据」的长度，完整响应长度 = 6（头） + n
	fullLength := 6 + int(n)

	// 步骤3：读取响应体（n个字节）
	body := make([]byte, n)
	nBody, err := io.ReadFull(c.conn, body)
	if err != nil {
		c.Receive_Response += 1
		log.Printf("ERROR modbus_tcp: 读取响应体失败：%v", err)
		c.conn_err = err
		_ = c.conn.Close()
		c.conn = nil

		if c.Receive_Response > 3 {
			c.tcp_sync.Unlock()
			reconnectErr := c.keepAliveConnect()
			c.tcp_sync.Lock()
			if reconnectErr != nil {
				return []byte{}, reconnectErr
			}
			return []byte{}, errors.New("触发重连，请重试请求")
		}
		return []byte{}, err
	}
	if nBody != int(n) {
		return []byte{}, fmt.Errorf("响应体长度不符，预期%d，实际%d", n, nBody)
	}

	// 步骤4：拼接完整响应（头+体）
	receive := append(header, body...)

	// 校验正确响应
	err = c.CheckTransactionID(*send, receive)
	if err != nil {
		log.Printf("WARN modbus_tcp: 事务ID校验失败: %v", err)
		// 事务ID不匹配可能是由于之前的请求残留或乱序，尝试清空缓冲区
		_, err1 := ClearTCPBuffer(c.conn, 20)
		if err1 != nil {
			log.Printf("ERROR modbus_tcp: 清空缓冲区失败: %v", err1)
		}

		// 校验失败通常视为一次通信异常，增加计数
		c.Receive_Response++
		if c.Receive_Response > 3 {
			c.tcp_sync.Unlock()
			reconnectErr := c.keepAliveConnect()
			c.tcp_sync.Lock()
			if reconnectErr != nil {
				return []byte{}, reconnectErr
			}
			return []byte{}, errors.New("触发重连，请重试请求")
		}

		return []byte{}, err
	}

	// 重置连续错误计数
	c.Receive_Response = 0

	// 后续你的逻辑不变（打印、校验ID等）
	if c.Data_Packet_Print_enable {
		fmt.Printf("接收:% x\n", receive[:fullLength]) // 这里可以直接用receive，因为已经是完整的
	}

	if len(receive) < fullLength {
		return []byte{}, fmt.Errorf("响应体长度不符，预期%d，实际%d", fullLength, len(receive))
	}

	return receive, nil
}

// 读取  00001至09999是离散输出(线圈)
// Start开始地址  Number个数
func (c *Connect_struct) Read__Coils_status(Device uint8, Start uint16, Number uint16) ([]bool, error) {
	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Number_byte := byte_convert.Convert_uint16_uint8([]uint16{Number}, byte_convert.AB)
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
		return []bool{}, errors.New("设备驱动没有这个功能码")
	}

	byte_value := r[9 : 9+int(r[8])]

	var value []bool
	for i := 0; i < len(byte_value); i++ {
		// a, err := byte_convert.Byte_Convert_1byte_8bool(byte_value[i])
		a := byte_convert.Convert_uint8_bool([]uint8{byte_value[i]})
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

	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Number_byte := byte_convert.Convert_uint16_uint8([]uint16{Number}, byte_convert.AB)

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
		return []bool{}, errors.New("设备驱动没有这个功能码")
	}

	byte_value := r[9 : 9+int(r[8])]
	var value []bool
	for i := 0; i < len(byte_value); i++ {
		a := byte_convert.Convert_uint8_bool([]uint8{byte_value[i]})
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

	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Number_byte := byte_convert.Convert_uint16_uint8([]uint16{Number}, byte_convert.AB)

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
		return []byte{}, errors.New("设备驱动没有这个功能码")
	}

	return r[9:], nil
}

// 读取 30001至39999是输入寄存器(通常是模拟量输入) 04功能码
// Start开始地址  Number个数
func (c *Connect_struct) Read__Input_register(Device uint8, Start uint16, Number uint16) ([]byte, error) {

	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Number_byte := byte_convert.Convert_uint16_uint8([]uint16{Number}, byte_convert.AB)

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
		return []byte{}, errors.New("设备驱动没有这个功能码")
	}

	return r[9:], nil
}

// 写入单个  00001至09999是离散输出(线圈)
// Start开始地址  Number个数
func (c *Connect_struct) Write_single__Coils_tatus(Device uint8, Start uint16, Value bool) error {
	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)

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

	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Value_Number := byte_convert.Convert_uint16_uint8([]uint16{uint16(len(Value))}, byte_convert.AB)

	Value_byte := byte_convert.Convert_bool_byte(Value)

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
	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)

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
	Start_byte := byte_convert.Convert_uint16_uint8([]uint16{Start}, byte_convert.AB)
	Value_Number := byte_convert.Convert_uint16_uint8([]uint16{Number}, byte_convert.AB)

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

func divideCeil(a, b int) int {
	result := a / b
	if a%b != 0 {
		result++
	}
	return result
}

// 先修改基础版 ClearTCPBuffer，使其返回丢弃的字节切片+字节数+错误
func ClearTCPBuffer(conn net.Conn, timeoutMs int) ([]byte, error) {
	// 2. 设置短超时用于清空缓冲区
	_ = conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs) * time.Millisecond))

	discardedData := make([]byte, 0) // 存储本次清空丢弃的所有数据
	buf := make([]byte, 1024)        // 每次读取的临时缓冲区

	for {
		n, err := conn.Read(buf)
		if err != nil {
			// 超时=缓冲区已空，正常退出（返回已读取的丢弃数据）
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("清空超时，本次共丢弃%d字节", len(discardedData))
				return discardedData, nil
			}
			// 非超时错误（如连接断开），返回已读取的数据+错误
			log.Printf("清空失败：%v，已丢弃%d字节", err, len(discardedData))
			return discardedData, err
		}

		if n == 0 {
			// 读取到0字节=连接关闭，退出
			log.Printf("连接关闭，清空结束，已丢弃%d字节", len(discardedData))
			break
		}

		// 将本次读取的字节追加到丢弃数据切片中
		discardedData = append(discardedData, buf[:n]...)
		log.Printf("本次读取并丢弃%d字节：% x", n, buf[:n])
	}

	return discardedData, nil
}
