package Modbus_Tcp

import (
	"main/IO/conv"
	"main/cloud"

	"fmt"
	"strconv"
	"strings"
)

const (
	// 正则表达式
	regex__address                 = `^(\d+;[1-9]\d*ms;[1-9]\d*ms;[1-9]\d*ms;[1-9]\d*ms;[1-9]\d*)$` // 地址
	regex__retryTimeout_index      = `^[+-]?((\d+(\.\d+)?)(ns|µs|us|ms|s|m|h))+$`                   // 重试间隔
	regex__connectTimeout_index    = regex__retryTimeout_index                                      // 连接超时
	regex__responseTimeout_index   = regex__retryTimeout_index                                      // 响应超时
	regex__delayBetweenPolls_index = regex__retryTimeout_index                                      // 轮询间隔
	regex__packetMax_index         = `^(0|[1-9]\d?|1[01]\d|12[0-5])$`                               // 组包最大长度

	// 配置项索引
	address_index           = 0 // 地址
	retryTimeout_index      = 1 // 重试间隔
	connectTimeout_index    = 2 // 连接超时
	responseTimeout_index   = 3 // 响应超时
	delayBetweenPolls_index = 4 // 轮询间隔
	packetMax_index         = 5 // 组包最大长度
)

// Drive_Config_Switch 解析驱动配置字符串 192.168.1.1;502;3s;200ms;1s;8
func Drive_Config_Switch(configStr string) (c Config_type, err error) {

	address, err := cloud.GetSplitStr(configStr, address_index, "IP")
	if err != nil {
		return
	}

	retryTimeout, err := cloud.GetSplitDuration(configStr, retryTimeout_index, "重试间隔")
	if err != nil {
		return
	}

	connectTimeout, err := cloud.GetSplitDuration(configStr, connectTimeout_index, "连接超时")
	if err != nil {
		return
	}

	responseTimeout, err := cloud.GetSplitDuration(configStr, responseTimeout_index, "响应超时")
	if err != nil {
		return
	}

	delayBetweenPolls, err := cloud.GetSplitDuration(configStr, delayBetweenPolls_index, "轮询间隔")
	if err != nil {
		return
	}

	// Packet_max 是数字，不是时间！
	packetMax, err := cloud.GetSplitInt(configStr, packetMax_index, "组包最大长度")
	if err != nil {
		err = fmt.Errorf("组包最大长度解析失败: %w", err)
		return
	}

	if packetMax%2 != 0 || packetMax == 0 {
		err = fmt.Errorf("组包数量必须是2的倍数且大于0, 当前值: %d", packetMax)
		return
	}

	// 4. 组装返回
	c = Config_type{
		Address:             address,
		Retry_timeout:       retryTimeout,
		Connect_timeout:     connectTimeout,
		Response_timeout:    responseTimeout,
		Delay_between_polls: delayBetweenPolls,
		Packet_max:          packetMax,
	}

	return
}

const (
	// 正则表达式
	regex__SlaveID    = `^(?:25[0-5]|2[0-4]\d|[01]?\d)$`                                         // 从机地址
	regex__Function   = `^[1-4]$`                                                                // Modbus功能码
	regex__Address    = `^(?:6553[0-5]|655[0-2]\d|65[0-4]\d{2}|6[0-4]\d{3}|[1-5]\d{4}|\d{1,4})$` // 寄存器地址
	regex__Byte_Order = ""                                                                       // 字节序
	regex__Type       = ""                                                                       // 数据类型

	// 配置项索引
	SlaveID_index    = 0 // 从机地址
	Function_index   = 1 // Modbus功能码（如3=读保持寄存器）
	Address_index    = 2 // 寄存器地址
	Byte_Order_index = 3 // 字节序（如"ABCD"表示大端）
	Type_index       = 4 // 数据类型（bool/int8/float32等）
)

func Point_Config_Switch(s string) (point Points_type, err error) {
	parts := strings.Split(s, ";")
	// 2. 必须是 4 段，否则格式错误
	if len(parts) < 5 {
		err = fmt.Errorf("点位配置格式错误，需要5段，实际%d段: %s", len(parts), s)
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
		err = fmt.Errorf("从机地址解析失败: %w", err)
		return
	}
	var Function int
	Function, err = strconv.Atoi(Function_str)
	if err != nil {
		err = fmt.Errorf("功能码解析失败: %w", err)
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
		err = fmt.Errorf("寄存器地址解析失败: %w", err)
		return
	}

	// 解析数据类型名称为类型编码
	typeCode, typeExists := conv.ValueTypeStringToInt(Type_str)
	if !typeExists {
		err = fmt.Errorf("无效的数据类型: %s", Type_str)
		return
	}

	var Child_Address int
	if (Function == 3 || Function == 4) && Type_str == "bool" {
		if len(Address_str2) > 1 {
			Child_Address, err = strconv.Atoi(Address_str2[1])
			if err != nil {
				err = fmt.Errorf("寄存器子地址解析失败: %w", err)
				return
			}
		}
	}

	var Byte_Order int
	if Function == 3 || Function == 4 {
		var exists bool
		Byte_Order, exists = conv.Byte_Value[Byte_Order_str]
		if !exists {
			err = fmt.Errorf("无效的字节序: %s", Byte_Order_str)
			return
		}
	}

	point = Points_type{
		SlaveID:       slaveID,       // 从机地址
		Function:      Function,      // Modbus功能码（如3=读保持寄存器）
		Address:       Address,       // 寄存器地址
		Type:          typeCode,      // 数据类型编码
		Child_Address: Child_Address, // 子地址（可选）
		Byte_Order:    Byte_Order,    // 字节序（如"ABCD"表示大端）
	}

	return
}

// func Point_Config_Switch_List(configs []string) (points []Points_type, err error)

// func Read_Callback_v(v []Read_Value_type) {
// 	fmt.Printf("Read_Callback >>>>>>>> \n%+v \n", v)
// }
