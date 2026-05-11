package Modbus_Tcp

import (
	"fmt"
	"log"
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
	Type_str := strings.TrimSpace(parts[3])       // 数据类型（bool/int8/float32等）
	Byte_Order_str := strings.TrimSpace(parts[4]) // 字节序（如"ABCD"表示大端）

	var slaveID int
	slaveID, err = strconv.Atoi(SlaveID_str)
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

	point = Points_type{
		SlaveID:       uint8(slaveID),       // 从机地址
		Function:      Function_str,         // Modbus功能码（如3=读保持寄存器）
		Address:       uint16(Address),      // 寄存器地址
		Type:          Type_str,             // 数据类型（bool/int8/float32等）
		Child_Address: uint8(Child_Address), // 子地址（可选）
		Byte_Order:    Byte_Order_str,       // 字节序（如"ABCD"表示大端）
	}

	return
}

// func Point_Config_Switch_List(configs []string) (points []Points_type, err error)
func New(config mqtt_rpc.IO_Points_Config_type) (err error) {

	var Drive_Config Config_type
	Drive_Config, err = Drive_Config_Switch(config.Drive.Config)
	if err != nil {
		log.Print(err)
		return
	}

	var Points_Config []Mysql_Points_type
	for _, pointStr := range config.Points {
		point, err := Point_Config_Switch(pointStr.Config)
		if err != nil {
			log.Printf("ERROR 解析点位配置失败: %v, 配置字符串: %s", err, pointStr.Config)
			continue
		}
		Points_Config = append(Points_Config, Mysql_Points_type{
			Id:         pointStr.Id,         // 点位id
			Drive_Id:   pointStr.Drive.Id,   // 驱动id唯一标识符
			Drive_Type: pointStr.Drive.Type, // 驱动类型
			Name:       pointStr.Tag,        // 点位名称
			RW_Cancel:  pointStr.RW_Cancel,  // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
			Value_Type: pointStr.Value_Type, // 输出类型
			Config:     point,
		})
	}

	c := Connect_struct{
		Drive: Mysql_Config_type{
			Id:     config.Drive.Id,   // 驱动id
			Type:   config.Drive.Type, // 驱动类型
			Name:   config.Drive.Name, // 驱动名称
			Config: Drive_Config,
		},
		Points: Points_Config,
	}

	fmt.Printf("Points |||||||||||||||| \n%+v \n\n", Points_Config)
	// 组包
	err = c.Packet()
	if err != nil {
		log.Printf("ERROR %v", err.Error())
		return
	}

	// 连接
	err = c.Connect()
	if err != nil {
		log.Printf("WARNING %v", err.Error())
	}

	fmt.Printf("Packets >>>>>>>> \n%+v \n", c.Packets)

	go c.Read_Continuous(func(g []IO_Collection_Value_type) {
		fmt.Printf("Read_Callback >>>>>>>> \n%+v \n", g)
	})

	return nil
}

// func Read_Callback_v(v []Read_Value_type) {
// 	fmt.Printf("Read_Callback >>>>>>>> \n%+v \n", v)
// }
