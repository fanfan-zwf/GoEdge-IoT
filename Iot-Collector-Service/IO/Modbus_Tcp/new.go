package Modbus_Tcp

import (
	"fmt"
	"log"
	"main/IO/byte_convert"
	"main/app/mqtt_rpc"
	"strconv"
	"strings"
	"time"
)

// Drive_Config_Switch 解析驱动配置字符串 192.168.1.1;502;3s;200ms;1s;8
func Drive_Config_Switch(configStr string) (c Config_type, err error) {
	// 1. 分割字符串
	parts := strings.Split(configStr, ";")
	if len(parts) < 6 {
		return Config_type{}, fmt.Errorf("ERROR 配置格式错误，字段数量不足，需要6段")
	}

	// 2. 定义解析函数（减少重复代码，优雅）
	parseDuration := func(index int, name string) (time.Duration, error) {
		d, err := time.ParseDuration(parts[index])
		if err != nil {
			return 0, fmt.Errorf("ERROR %s 解析失败: %w", name, err)
		}
		return d, nil
	}

	// 3. 逐个解析（一一对应，不再读错）
	ip := parts[0]

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		err = fmt.Errorf("端口解析失败: %w", err)
		return
	}

	retryTimeout, err := parseDuration(2, "重试间隔")
	if err != nil {
		return
	}

	connectTimeout, err := parseDuration(3, "连接超时")
	if err != nil {
		return
	}

	responseTimeout, err := parseDuration(4, "响应超时")
	if err != nil {
		return
	}

	delayBetweenPolls, err := parseDuration(5, "轮询间隔")
	if err != nil {
		return
	}

	// Packet_max 是数字，不是时间！
	packetMax, err := strconv.Atoi(parts[6])
	if err != nil {
		err = fmt.Errorf("ERROR 组包最大长度不是数字%w", err)
		return
	}

	if packetMax%2 != 0 || packetMax == 0 {
		err = fmt.Errorf("ERROR 组包数量必须是2的倍数并且不能对于0 组包: %d", packetMax)
		return
	}

	// 4. 组装返回
	c = Config_type{
		Ip:                  ip,
		Port:                uint16(port),
		Retry_timeout:       retryTimeout,
		Connect_timeout:     connectTimeout,
		Response_timeout:    responseTimeout,
		Delay_between_polls: delayBetweenPolls,
		Packet_max:          uint8(packetMax),
	}

	return
}
func Point_Config_Switch(s string) (point Points_type, err error) {
	parts := strings.Split(s, ";")
	// 2. 必须是 4 段，否则格式错误
	if len(parts) < 5 {
		err = fmt.Errorf("格式错误，需要 4 段，实际 %d 段", len(parts))
		return
	}

	SlaveID_str := strings.TrimSpace(parts[0])    // 从机地址
	Function_str := strings.TrimSpace(parts[1])   // Modbus功能码（如3=读保持寄存器）
	Address_str := strings.TrimSpace(parts[2])    // 寄存器地址
	Byte_Order_str := strings.TrimSpace(parts[3]) // 字节序（如"ABCD"表示大端）
	Type_str := strings.TrimSpace(parts[4])       // 数据类型（bool/int8/float32等）

	var slaveID int
	slaveID, err = strconv.Atoi(SlaveID_str)
	if err != nil {
		err = fmt.Errorf("ERROR 从机地址不是数字%w", err)
		return
	}

	var Function int
	Function, err = strconv.Atoi(Function_str)
	if err != nil {
		err = fmt.Errorf("ERROR 从机地址不是数字%w", err)
		return
	}

	Address_str2 := strings.Split(Address_str, ".")
	if len(Address_str2) == 0 {
		err = fmt.Errorf("ERROR 不存在寄存器地址 %d", len(Address_str2))
		return
	}

	var Address int
	Address, err = strconv.Atoi(Address_str2[0])
	if err != nil {
		err = fmt.Errorf("ERROR 从机地址不是数字%w", err)
		return
	}

	var Child_Address int
	if len(Address_str2) > 1 {
		Child_Address, err = strconv.Atoi(Address_str2[1])
		if err != nil {
			err = fmt.Errorf("ERROR 寄存器子地址不是数字%w", err)
			return
		}
	}

	var byteOrderMap = map[string]int{
		"AB":   byte_convert.AB,
		"BA":   byte_convert.BA,
		"ABCD": byte_convert.ABCD,
		"ABDC": byte_convert.ABDC,
		"BACD": byte_convert.BACD,
		"DCBA": byte_convert.DCBA,
	}

	// 解析字节序
	Byte_Order, exists := byteOrderMap[Byte_Order_str]
	if !exists {
		err = fmt.Errorf("字节序格式错误: %s", Byte_Order_str)
		return
	}

	point = Points_type{
		SlaveID:       uint8(slaveID),       // 从机地址
		Function:      uint8(Function),      // Modbus功能码（如3=读保持寄存器）
		Address:       uint16(Address),      // 寄存器地址
		Type:          Type_str,             // 数据类型（bool/int8/float32等）
		Child_Address: uint8(Child_Address), // 子地址（可选）
		Byte_Order:    Byte_Order,           // 字节序（如"ABCD"表示大端）
	}

	return
}

// func Point_Config_Switch_List(configs []string) (points []Points_type, err error)
func New(config mqtt_rpc.IO_Points_Config_type) error {

	// c := Connect_struct{Drive_Id: config.Drive.Id, AllConfig: config}

	c, err := NewTCPClient(config)
	if err != nil {
		log.Printf("ERROR:%s", err)
	}
	err = c.Connect()
	if err != nil {
		log.Printf("ERROR:%s", err)
	}

	fmt.Printf("\n%+v >>>>>>>\n", c)
	return nil
}
