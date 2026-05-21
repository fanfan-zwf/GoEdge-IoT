/*
* 日期: 2025.5.29 PM10:17
* 作者: 范范zwf
* 作用: modbus tcp组包
 */

package Modbus_Tcp

import (
	"main/IO/byte_util"
	"main/Init"

	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"golang.org/x/exp/constraints"
)

var Err_Prediction_type_not = errors.New("预测类型不存在")

var Config Init.Config_type = Init.Config

type middle_type map[uint8]map[string][]Packet_Address_type

/*
 *****************组包*****************
 */

// 这个结构体是用来组包的中间结构体
type Packet_Address_type struct {
	Address   uint16
	Points_Id uint
}

// 定义一个结构体
type Packet_struct struct {
	// middle     middle_type // 点位合并
	Packets    Packet_type // 组包后的
	Packet_max uint8       // 组包个数
}

// 定义接口
type Packet_interface interface {
	// 组包把所有点位放到数组里
	Packet_Address_merge(Points []Mysql_Points_type) (middle_type, error)

	// 这里是要对已经遍历出来的点位进行组包
	Packet_Address_process(middle middle_type) ([]Packet_type, error)

	// 这里增加要对已经合并的点位进行处理
	// 传入功能码和寄存器数组
	Packet_Address_register_add(byte_Number uint8, Address []Packet_Address_type) (Packet_type, error)

	// 这里要对已经合并的点位进行处理
	// 传入功能码
	Packet_Address_register(SlaveID uint8, Function string, Address []Packet_Address_type) ([]Packet_type, error)

	// 组包顺序判断
	Packet_Address_register_order(Address []Packet_Address_type) error
}

// 类型字节数量输出
var type_byte = map[string]int{
	"bool":    1,
	"int16":   1,
	"uint16":  1,
	"int32":   2,
	"uint32":  2,
	"float32": 2,
}

// 写入
func Packet_Points(Points_Id uint, Address uint16, Type string) ([]Packet_Address_type, error) {
	var Address_Packet []Packet_Address_type
	// 增加点位置临时变量
	v, ok := type_byte[Type]
	if !ok {
		return []Packet_Address_type{}, errors.New("not byte")
	}

	for k := 0; k < v; k++ {
		Address_Packet = append(Address_Packet, Packet_Address_type{
			Points_Id: Points_Id,
			Address:   Address + uint16(k),
		})
	}

	if len(Address_Packet) >= 1 {
		return Address_Packet, nil
	}

	return []Packet_Address_type{}, errors.New("not executed")
}

// 组包把所有点位放到数组里
func (p *Packet_struct) Packet_Address_merge(Points []Mysql_Points_type) (middle_type, error) {
	middle := make(middle_type)
	for _, v := range Points {

		if !(v.RW_Cancel == "R" || v.RW_Cancel == "W/R") {
			log.Print("ERROR 这个不是一个可读的点位")
			continue
		}

		if v.Drive_Type != "Modbus_Tcp" {
			log.Print("ERROR 这个点位驱动类型不正确")
			continue
		}

		// 判断设备集合是否存在不存在增加
		_, ok := middle[v.Config.SlaveID]
		if !ok {
			// key 不存在
			middle[v.Config.SlaveID] = map[string][]Packet_Address_type{}
		}
		// 判断功能码集合是否存在不存在增加
		_, ok = middle[v.Config.SlaveID][v.Config.Function]
		if !ok {
			// key 不存在
			middle[v.Config.SlaveID][v.Config.Function] = []Packet_Address_type{}
		}

		// 增加点位到中间变量Packets
		Packet_Address, err := Packet_Points(v.Id, v.Config.Address, v.Config.Type)
		if err != nil {
			break
		}

		// 增加临时变量
		middle[v.Config.SlaveID][v.Config.Function] = append(middle[v.Config.SlaveID][v.Config.Function], Packet_Address...)

		// 排序
		sort.Slice(middle[v.Config.SlaveID][v.Config.Function], func(i, j int) bool {
			if middle[v.Config.SlaveID][v.Config.Function][i].Address != middle[v.Config.SlaveID][v.Config.Function][j].Address {
				return middle[v.Config.SlaveID][v.Config.Function][i].Address < middle[v.Config.SlaveID][v.Config.Function][j].Address
			}
			return middle[v.Config.SlaveID][v.Config.Function][i].Points_Id < middle[v.Config.SlaveID][v.Config.Function][j].Points_Id
		})

	}

	// fmt.Print("middle排序后:", middle, "\n")

	return middle, nil
}

// 判断数组是否一致
func AllSame[T comparable](slice []T) bool {
	if len(slice) == 0 {
		return true // 空切片视为所有元素相同
	}

	first := slice[0]
	for _, v := range slice[1:] {
		if v != first {
			return false
		}
	}
	return true
}

// 判断数组是否递增
func IsIncreasing[T constraints.Ordered](slice []T) bool {
	for i := 1; i < len(slice); i++ {
		if slice[i] <= slice[i-1] {
			return false
		}
	}
	return true
}

// 组包顺序判断
func (p *Packet_struct) Packet_Address_register_order(Address []Packet_Address_type) error {
	var Points_Id_s []uint
	var Addresss []uint16
	for _, v := range Address {
		Points_Id_s = append(Points_Id_s, v.Points_Id)
		Addresss = append(Addresss, v.Address)
	}
	if !AllSame(Points_Id_s) {
		return errors.New("传入的点位数组,点位id不一致")
	}

	if len(Address) > 1 { // 多个寄存器
		if !IsIncreasing(Addresss) {
			return fmt.Errorf("传入的点位数组,寄存器不是递增")
		}
	}
	return nil
}

// 这里增加要对已经合并的点位进行处理
// 传入功能码和寄存器数组
func (p *Packet_struct) Packet_Address_register_add(Packet *Packet_type, Address []Packet_Address_type) error {
	if len(Address) == 0 {
		return errors.New("长度nil")
	}

	// 排序
	sort.Slice(Address, func(i, j int) bool {
		return Address[i].Address < Address[j].Address
	})

	var Addresss []uint16
	for _, v := range Address {
		if v.Points_Id != Address[0].Points_Id {
			log.Print("ERROR 不是同一个点位,这个是一个非常一严重的错误")
			return errors.New("不一个同一个点位")
		}
		Addresss = append(Addresss, v.Address)
	}

	// 判断包是否有这个点位
	if (*Packet).Start_Address <= Addresss[0] && (Addresss[0] < (*Packet).Start_Address+(*Packet).Number_Address) {

		for _, v := range (*Packet).PointsId {
			if v == Address[0].Points_Id {
				return nil
			}
		}

	}

	// 判断是否可以一起组包
	if (((*Packet).Start_Address+(*Packet).Number_Address)+1 <= Addresss[0]) && ((*Packet).Start_Address <= Addresss[0]) {
		return errors.New("不可以放在一起进行组包")
	}

	if (*Packet).Number_Address+uint16(len(Addresss)) > uint16(p.Packet_max) {
		return errors.New("组包长度过长")
	}

	(*Packet).Number_Address = (*Packet).Number_Address + uint16(len(Addresss))
	(*Packet).PointsId = append((*Packet).PointsId, Address[0].Points_Id)

	return nil
}

// 这里要对已经合并的点位进行处理
// 传入功能码
func (p *Packet_struct) Packet_Address_register(SlaveID uint8, Function string, Address []Packet_Address_type) ([]Packet_type, error) {
	var (
		Packet_array    []Packet_type
		Packet          Packet_type
		Before_PointsId uint
		Address_array   []Packet_Address_type
	)
	// 排序
	sort.Slice(Address, func(i, j int) bool {
		if Address[i].Address != Address[j].Address {
			return Address[i].Address < Address[j].Address
		}
		return Address[i].Points_Id < Address[j].Points_Id
	})

	for i, v := range Address {
		if Before_PointsId == 0 && i == 0 {
			Address_array = append(Address_array, v)
			Before_PointsId = v.Points_Id
			Packet = Packet_type{
				SlaveID:        SlaveID,             // 从机地址
				Function:       Function,            // 功能码
				Start_Address:  v.Address,           // 开始地址
				Number_Address: 1,                   // 地址数量
				PointsId:       []uint{v.Points_Id}, // 这个包
			}

			continue
		}

		if Before_PointsId == v.Points_Id { // 还是这个点位
			Address_array = append(Address_array, v)
		}

		if Before_PointsId != v.Points_Id || i >= len(Address)-1 {
			// Addresss 这个是一个点位的所有寄存器

			err := p.Packet_Address_register_order(Address_array)
			if err != nil {
				log.Print("ERROR 这是一个非常严重的错误", err.Error())
				return []Packet_type{}, err
			}

			err = p.Packet_Address_register_add(&Packet, Address_array)

			if err != nil { // 增加失败
				Packet_array = append(Packet_array, Packet)

				Packet = Packet_type{
					SlaveID:        SlaveID,                            // 从机地址
					Function:       Function,                           // 功能码
					Start_Address:  Address_array[0].Address,           // 开始地址
					Number_Address: uint16(len(Address_array)),         // 地址数量
					PointsId:       []uint{Address_array[0].Points_Id}, // 这个包
				}

			}
			Address_array = []Packet_Address_type{v}
		}

		if i >= len(Address)-1 {
			if Before_PointsId == v.Points_Id { // 还是这个点位
				Address_array = append(Address_array, v)
			}

			// Addresss 这个是一个点位的所有寄存器

			err := p.Packet_Address_register_order(Address_array)
			if err != nil {
				log.Print("ERROR 这是一个非常严重的错误", err.Error())
				return []Packet_type{}, err
			}

			err = p.Packet_Address_register_add(&Packet, Address_array)

			if err != nil { // 增加失败
				Packet_array = append(Packet_array, Packet)

				Packet = Packet_type{
					SlaveID:        SlaveID,                            // 从机地址
					Function:       Function,                           // 功能码
					Start_Address:  Address_array[0].Address,           // 开始地址
					Number_Address: uint16(len(Address_array)),         // 地址数量
					PointsId:       []uint{Address_array[0].Points_Id}, // 这个包
				}

			}

		}

		Before_PointsId = v.Points_Id

	}
	Packet_array = append(Packet_array, Packet)

	return Packet_array, nil
}

// 这里是要对已经遍历出来的点位进行组包
func (p *Packet_struct) Packet_Address_process(middle middle_type) ([]Packet_type, error) {
	var Packets []Packet_type

	// 循环设备id
	for SlaveID, Function := range middle {

		// 循环功能码
		for Function, Address := range Function {
			// if Function == "03" || Function == "04" {
			Packet, err := p.Packet_Address_register(SlaveID, Function, Address)
			if err != nil {
				return []Packet_type{}, err
			}
			Packets = append(Packets, Packet...)
			// }

		}

	}

	return Packets, nil
}

/*
 *****************这里还是Connect_struct实例*****************
 */

// 接受数据进行拆包

// 查询点位信息
func (c *Connect_struct) query_points_config(Points_Id uint) (Mysql_Points_type, error) {
	for _, v := range c.Points {
		if v.Id == Points_Id {
			return v, nil
		}
	}
	return Mysql_Points_type{}, fmt.Errorf("no point exists")
}

// 线圈读取正确处理
func (c *Connect_struct) read_status_ok(value []bool, Packet Packet_type) []IO_Collection_Value_type {

	var value_array []IO_Collection_Value_type

	for _, Point_Id := range Packet.PointsId {
		mysql_Points, err := c.query_points_config(Point_Id)
		if err != nil {
			log.Print("ERROR 配置不存在的点位")
			continue
		}
		if !(mysql_Points.RW_Cancel == "R" || mysql_Points.RW_Cancel == "W/R") {
			log.Print("ERROR 这个不是一个可读的点位")
			continue
		}

		if mysql_Points.Drive_Type != "Modbus_Tcp" {
			log.Print("ERROR 这个点位驱动类型不正确")
			continue
		}

		if mysql_Points.Value_Type == "" {
			value_array = append(value_array, IO_Collection_Value_type{
				Points_Id:  Point_Id,                        // 点位id
				Msg:        "值类型错误",                         // 状态
				Value_Type: "",                              // 值类型
				Value:      nil,                             // 值
				Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
			})
			log.Printf("ERROR 点位id:%d 值类型错误", Point_Id)
			continue
		}

		value_array = append(value_array, IO_Collection_Value_type{
			Points_Id:  Point_Id,                                                // 点位id
			Msg:        "ok",                                                    // 状态
			Value_Type: mysql_Points.Value_Type,                                 // 值类型
			Value:      value[mysql_Points.Config.Address-Packet.Start_Address], // 值
			Time:       time.Now().Format(Init.RFC_FAN),                         // 时间戳
		})
	}

	return value_array
}

func Packet_confirm(Packet []Packet_type, Points []Mysql_Points_type) error {
	var (
		allid_array    []uint
		packetid_array []uint
		exist          bool
		no_exist       []uint
	)
	for _, v := range Points {
		allid_array = append(allid_array, v.Id)
	}

	for _, p := range Packet {
		packetid_array = append(packetid_array, p.PointsId...)
	}

	for _, allid := range allid_array {
		exist = false
		for _, packetid := range packetid_array {
			if allid == packetid {
				exist = true
				continue
			}
		}
		if !exist {
			no_exist = append(no_exist, allid)
		}

	}

	if len(no_exist) != 0 {
		return fmt.Errorf("modbus_tcp 点位% d 未组包", no_exist)
	}

	return nil
}

// 线圈读取错误处理
func (c *Connect_struct) read_status_err(Msg string, PointsId []uint) []IO_Collection_Value_type {
	var value_array []IO_Collection_Value_type
	for _, Point_Id := range PointsId {
		mysql_Points, err := c.query_points_config(Point_Id)
		if err != nil {
			log.Print("ERROR 配置不存在的点位")
			continue
		}
		if !(mysql_Points.RW_Cancel == "R" || mysql_Points.RW_Cancel == "W/R") {
			log.Print("ERROR 这个不是一个可读的点位")
			continue
		}

		if mysql_Points.Drive_Type != "Modbus_Tcp" {
			log.Print("ERROR 这个点位驱动类型不正确")
			continue
		}
		if mysql_Points.Value_Type == "" {
			value_array = append(value_array, IO_Collection_Value_type{
				Points_Id:  Point_Id,                        // 点位id
				Msg:        "值类型错误",                         // 状态
				Value_Type: "",                              // 值类型
				Value:      nil,                             // 值
				Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
			})
			log.Printf("ERROR 点位id:%d 值类型错误", Point_Id)
			continue
		}

		value_array = append(value_array, IO_Collection_Value_type{
			Points_Id:  Point_Id,                        // 点位id
			Msg:        Msg,                             // 状态
			Value_Type: mysql_Points.Value_Type,         // 值类型
			Value:      nil,                             // 值
			Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
		})
	}
	return value_array
}

func divideCeil(a, b int) int {
	result := a / b
	if a%b != 0 {
		result++
	}
	return result
}

// 模拟量读取错误处理
func (c *Connect_struct) read_register_err(Msg string, PointsId []uint) []IO_Collection_Value_type {
	var value_array []IO_Collection_Value_type
	for _, Point_Id := range PointsId {
		mysql_Points, err := c.query_points_config(Point_Id)
		if err != nil {
			log.Print("ERROR 配置不存在的点位")
			continue
		}
		if !(mysql_Points.RW_Cancel == "R" || mysql_Points.RW_Cancel == "W/R") {
			log.Print("ERROR 这个不是一个可读的点位")
			continue
		}

		if mysql_Points.Drive_Type != "Modbus_Tcp" {
			log.Print("ERROR 这个点位驱动类型不正确")
			continue
		}

		if mysql_Points.Value_Type == "" {
			value_array = append(value_array, IO_Collection_Value_type{
				Points_Id:  Point_Id,                        // 点位id
				Msg:        "值类型错误",                         // 状态
				Value_Type: "",                              // 值类型
				Value:      nil,                             // 值
				Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
			})
			log.Printf("ERROR 点位id:%d 值类型错误", Point_Id)
		}

		value_array = append(value_array, IO_Collection_Value_type{
			Points_Id:  Point_Id,                        // 点位id
			Msg:        Msg,                             // 状态
			Value_Type: mysql_Points.Value_Type,         // 值类型
			Value:      nil,                             // 值
			Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
		})
	}

	return value_array
}

// 模拟量读取正确处理
func (c *Connect_struct) read_register_ok(value []byte, Packet Packet_type) []IO_Collection_Value_type {
	var value_array []IO_Collection_Value_type

	for _, Point_Id := range Packet.PointsId {
		mysql_Points, err := c.query_points_config(Point_Id)
		if err != nil {
			log.Print("ERROR 配置不存在的点位")
			continue
		}
		if !(mysql_Points.RW_Cancel == "R" || mysql_Points.RW_Cancel == "W/R") {
			log.Print("ERROR 这个不是一个可读的点位")
			continue
		}

		if mysql_Points.Drive_Type != "Modbus_Tcp" {
			log.Print("ERROR 这个点位驱动类型不正确")
			continue
		}

		Type_length, ok := type_byte[mysql_Points.Config.Type]
		if !ok {
			log.Print("ERROR no byte")
			continue
		}

		sv := int(mysql_Points.Config.Address-Packet.Start_Address) * 2
		nv := (int(mysql_Points.Config.Address-Packet.Start_Address) + Type_length) * 2

		rv, Type, err := read_register_byte_Convert(mysql_Points, value[sv:nv])
		if err != nil {
			log.Print("ERROR ", err.Error())
			continue
		}

		// if Type != mysql_Points.Value_Type {
		// 	value_array = append(value_array, IO_Collection_Value_type{
		// 		Points_Id:  Point_Id,                        // 点位id
		// 		Msg:        "值类型错误",                         // 状态
		// 		Value_Type: "",                              // 值类型
		// 		Value:      nil,                             // 值
		// 		Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
		// 	})
		// 	log.Printf("ERROR 点位id:%d 值类型错误", Point_Id)
		// 	continue
		// }

		value_array = append(value_array, IO_Collection_Value_type{
			Points_Id:  Point_Id,                        // 点位id
			Msg:        "ok",                            // 状态
			Value_Type: Type,                            // 值类型
			Value:      rv,                              // 值
			Time:       time.Now().Format(Init.RFC_FAN), // 时间戳
		})
	}

	return value_array
}

// 整数增加小数点,转化浮点数
func addDecimalPoint(num int, decimalPlaces int) float64 {
	divisor := 1.0
	for i := 0; i < decimalPlaces; i++ {
		divisor *= 10
	}
	return float64(num) / divisor
}

/*
**************模拟量字节值转换**************
 */
// 这个后面再优化吧,这里太复杂了
// 这个后面再优化吧,这里太复杂了
// 这个后面再优化吧,这里太复杂了
func read_register_byte_Convert(Points_config Mysql_Points_type, v []byte) (interface{}, string, error) {
	Type_length, ok := type_byte[Points_config.Config.Type]
	if !ok {
		return nil, Points_config.Value_Type, errors.New("no byte")
	}

	if len(v) != (Type_length * 2) {
		return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)

	}

	switch Points_config.Config.Type {
	case "bool":
		bools := byte_util.Convert_uint8_bool([]byte{v[1], v[0]}, Points_config.Config.Byte_Order)
		return bools[Points_config.Config.Child_Address], "bool", nil
	case "int16":
		intv := byte_util.Convert_uint8_int16([]byte{v[1], v[0]}, Points_config.Config.Byte_Order)
		if len(intv) == 0 {
			return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)
		}
		return int(intv[0]), "int", nil
	case "uint16":
		uintv := byte_util.Convert_uint8_uint16([]byte{v[1], v[0]}, Points_config.Config.Byte_Order)
		if len(uintv) == 0 {
			return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)
		}
		return int(uintv[0]), "uint", nil
	case "int32":
		intv := byte_util.Convert_uint8_int32([]byte{v[0], v[1], v[2], v[3]}, Points_config.Config.Byte_Order)
		if len(intv) == 0 {
			return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)
		}
		return int(intv[0]), "int", nil
	case "uint32":
		uintv := byte_util.Convert_uint8_uint32([]byte{v[0], v[1], v[2], v[3]}, Points_config.Config.Byte_Order)
		if len(uintv) == 0 {
			return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)
		}
		return int(uintv[0]), "uint", nil
	case "float32":
		floatv := byte_util.Convert_uint8_float32([]byte{v[0], v[1], v[2], v[3]}, Points_config.Config.Byte_Order)
		if len(floatv) == 0 {
			return nil, Points_config.Value_Type, fmt.Errorf("error length 类型长度%d,byte值长度%d", Type_length, len(v)*2)
		}
		return float64(floatv[0]), "float", nil
	}

	return nil, Points_config.Value_Type, errors.New("no Config Type")
}

/*
**************写线圈值转换**************
 */

// 01功能码
func (c *Connect_struct) Write_register_Coils_tatus(point_config Mysql_Points_type, value any) (err error) {
	if point_config.Drive_Type != "modbus_tcp" {
		return fmt.Errorf("modbus_tcp写值错误:点位设备类型, 点位id%d,驱动%s", point_config.Id, point_config.Drive_Type)
	}
	if !(point_config.RW_Cancel == "W/R" || point_config.RW_Cancel == "W") {
		return fmt.Errorf("modbus_tcp写值错误:禁止写, 点位id%d,读写模式%s", point_config.Id, point_config.RW_Cancel)
	}

	if point_config.Config.Function != "01" {
		return fmt.Errorf("modbus_tcp写值错误:功能码, 点位id%d,功能码%s", point_config.Id, point_config.Config.Function)
	}

	if point_config.Config.Type != "bool" {
		return fmt.Errorf("modbus_tcp写值错误:值类型, 点位id%d,值类型%s", point_config.Id, point_config.Config.Type)
	}

	if point_config.Value_Type != "bool" {
		return fmt.Errorf("modbus_tcp写值错误:输出值类型, 点位id%d,输出值类型%s", point_config.Id, point_config.Value_Type)
	}

	v, ok := value.(bool)
	if !ok {
		return fmt.Errorf("输入类型错误")
	}

	return c.Write_single__Coils_tatus(point_config.Config.SlaveID, point_config.Config.Address-1, v)
}

// 03功能码 写入

// 判断浮点数小数点移动后是否能安全转换为int
func CanConvertAfterScaling(f float64, decimalPlaces int) (bool, error) {
	// 移动小数点：乘以10的n次方
	scaleFactor := math.Pow10(decimalPlaces)
	scaledValue := f * scaleFactor

	// 1. 检查是否为有效数字
	if math.IsNaN(scaledValue) {
		return false, fmt.Errorf("结果为NaN，无法转换")
	}
	if math.IsInf(scaledValue, 0) {
		return false, fmt.Errorf("结果超出范围，无法转换")
	}

	// 2. 检查是否在int范围内
	if scaledValue < float64(math.MinInt) || scaledValue > float64(math.MaxInt) {
		return false, fmt.Errorf("值%.2f超出int范围[%d, %d]",
			scaledValue, math.MinInt, math.MaxInt)
	}

	// 3. 检查是否为整数（考虑浮点数精度）
	if !isEffectivelyInteger(scaledValue) {
		return false, fmt.Errorf("值%.10f不是有效整数", scaledValue)
	}

	return true, nil
}

// 考虑浮点数精度判断是否为整数
func isEffectivelyInteger(f float64) bool {
	// 使用小误差容限处理浮点数精度问题
	_, fractional := math.Modf(f)
	return math.Abs(fractional) < 1e-12
}

// 安全转换函数
func SafeConvertAfterScaling(f float64, decimalPlaces int) (int, error) {
	canConvert, err := CanConvertAfterScaling(f, decimalPlaces)
	if !canConvert {
		return 0, err
	}

	scaleFactor := math.Pow10(decimalPlaces)
	scaledValue := f * scaleFactor

	// 四舍五入后转换，避免浮点数精度问题
	rounded := math.Round(scaledValue)
	return int(rounded), nil
}

func (c *Connect_struct) Write_register_Input_register(point_config Mysql_Points_type, value any) error {
	if point_config.Drive_Type != "modbus_tcp" {
		return fmt.Errorf("modbus_tcp写值错误:点位设备类型, 点位id%d,驱动%s", point_config.Id, point_config.Drive_Type)
	}
	if !(point_config.RW_Cancel == "W/R" || point_config.RW_Cancel == "W") {
		return fmt.Errorf("modbus_tcp写值错误:禁止写, 点位id%d,读写模式%s", point_config.Id, point_config.RW_Cancel)
	}

	// bool
	if point_config.Config.Type == "bool" {
		if point_config.Value_Type != "bool" {
			return fmt.Errorf("输出类型计算错误,应该是bool,而是%s", point_config.Value_Type)
		}
		value_bool, ok := value.(bool)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}
		v, err := c.Read__Holding_register(point_config.Config.SlaveID, point_config.Config.Address-1, 1)
		if err != nil {
			return err
		}

		bools := byte_util.Convert_uint8_bool([]byte{v[0], v[1]}, point_config.Config.Byte_Order)
		if len(bools) < 16 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(v))
		}

		bools[point_config.Config.Child_Address] = value_bool
		Write_value := byte_util.Convert_bool_byte(bools, point_config.Config.Byte_Order)
		if len(Write_value) < 2 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}

		return c.Write_single__Input_register(point_config.Config.SlaveID, point_config.Config.Address-1, [2]byte{Write_value[0], Write_value[1]})
	}

	// uint16 无小数点
	if point_config.Config.Type == "uint16" {
		if point_config.Value_Type != "uint" {
			return fmt.Errorf("输出类型计算错误,应该是uint,而不是%s", point_config.Value_Type)
		}

		value_uint, ok := value.(uint)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}
		Write_value := byte_util.Convert_uint16_uint8([]uint16{uint16(value_uint)}, point_config.Config.Byte_Order)
		if len(Write_value) < 2 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}
		return c.Write_single__Input_register(
			point_config.Config.SlaveID, point_config.Config.Address-1, [2]byte{
				Write_value[1], Write_value[0],
			})
	}

	// int16 无小数点
	if point_config.Config.Type == "int16" {
		if point_config.Value_Type != "int" {
			return fmt.Errorf("输出类型计算错误,应该是int,而不是%s", point_config.Value_Type)
		}

		value_uint, ok := value.(int)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}
		Write_value := byte_util.Convert_int16_uint8([]int16{int16(value_uint)}, point_config.Config.Byte_Order)
		if len(Write_value) < 2 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}
		return c.Write_single__Input_register(
			point_config.Config.SlaveID, point_config.Config.Address-1, [2]byte{
				Write_value[1], Write_value[0],
			})
	}

	// uint32 无小数点
	if point_config.Config.Type == "uint32" {
		if point_config.Value_Type != "uint" {
			return fmt.Errorf("输出类型计算错误,应该是uint,而不是%s", point_config.Value_Type)
		}

		value_uint, ok := value.(uint)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}
		Write_value := byte_util.Convert_uint32_uint8([]uint32{uint32(value_uint)}, point_config.Config.Byte_Order)
		if len(Write_value) < 4 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}
		return c.Write_many__Input_register(
			point_config.Config.SlaveID, point_config.Config.Address-1, 2, []byte{
				Write_value[0], Write_value[1], Write_value[2], Write_value[3],
			})
	}

	// int32 无小数点
	if point_config.Config.Type == "int32" {
		if point_config.Value_Type != "int" {
			return fmt.Errorf("输出类型计算错误,应该是int,而不是%s", point_config.Value_Type)
		}

		value_int, ok := value.(int)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}

		Write_value := byte_util.Convert_int32_uint8([]int32{int32(value_int)}, point_config.Config.Byte_Order)
		if len(Write_value) < 4 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}
		return c.Write_many__Input_register(
			point_config.Config.SlaveID, point_config.Config.Address-1, 2, []byte{
				Write_value[0], Write_value[1], Write_value[2], Write_value[3],
			})
	}

	// float32
	if point_config.Config.Type == "float32" {
		if point_config.Value_Type != "float" && point_config.Value_Type != "float32" {
			return fmt.Errorf("输出类型计算错误,应该是float32,而不是%s", point_config.Value_Type)
		}

		value_float, ok := value.(float64)
		if !ok {
			return fmt.Errorf("输入类型错误")
		}

		Write_value := byte_util.Convert_float32_uint8([]float32{float32(value_float)}, point_config.Config.Byte_Order)
		if len(Write_value) < 4 {
			return fmt.Errorf("error length 类型长度%d,byte值长度%d", 1, len(Write_value))
		}
		return c.Write_many__Input_register(
			point_config.Config.SlaveID, point_config.Config.Address-1, 2, []byte{
				Write_value[0], Write_value[1], Write_value[2], Write_value[3],
			})
	}

	return fmt.Errorf("modbus_tcp写值错误:未知类型, 点位id%d,值类型%s", point_config.Id, point_config.Config.Function)

}
