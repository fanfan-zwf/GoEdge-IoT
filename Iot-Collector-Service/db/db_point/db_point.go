/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	my_redis "main/db/redis"
	"time"

	"encoding/json"
	"fmt"
	"log"
	"sync"
)

var (
	Err_Publisher_Close = fmt.Errorf("Close")
)

func init() {

}

/*
******************IO数据采集******************
 */

// IO数据采集结构体
type Db_Value_type struct {
	Id   uint   // 点位id
	Tag  string // 点位标识
	Msg  string // 状态
	Type string // 值类型

	Value any // 值
	Time  time.Time
}

type Db_func func([]Db_Value_type) error

var (
	db_value_list []*Db_func
	Db_mu         sync.Mutex
)

// IO数据采集 发布 发送
func Db_Publisher(value []Db_Value_type) (err error) {
	if len(value) == 0 {
		return nil
	}

	for _, v := range value {
		if !(v.Type == "bool" ||
			v.Type == "int" ||
			v.Type == "uint" ||

			v.Type == "int8" ||
			v.Type == "uint8" ||

			v.Type == "int16" ||
			v.Type == "uint16" ||

			v.Type == "int32" ||
			v.Type == "uint32" ||

			v.Type == "int64" ||
			v.Type == "uint64" ||

			v.Type == "float" ||
			v.Type == "float32" ||
			v.Type == "float64") {
			err = fmt.Errorf("ERROR 点位型不正确 %v", v.Type)
			return err
		}
		if v.Id == 0 {
			err = fmt.Errorf("ERROR 点位id不正确 %d", v.Id)
			return err
		}

		if v.Tag == "" {
			err = fmt.Errorf("ERROR 点位标识不正确 %d", v.Id)
			return err
		}

		var consistency bool
		consistency, err = Drive_Type_Map__Consistency(
			v.Tag,
			"",
			"",
			v.Type,
		)

		if err != nil {
			log.Print(err)
			err = nil
			continue
		}

		if !consistency {
			log.Printf("ERROR 接收到信息和配置不一致")
			continue
		}

	}

	var toDelete []int
	for i, v := range db_value_list {
		if v == nil {
			toDelete = append(toDelete, i)
			log.Printf("WARNING 关闭一个 IO采集 的订阅者 索引:%d", i)
			continue
		}
		err = (*v)(value)
		if err == Err_Publisher_Close {
			db_value_list[i] = nil
		} else if err != nil {
			log.Print(err)
			return err
		}

	}

	if len(toDelete) == 0 {
		return nil
	}

	// 统一删除需要关闭的订阅者（倒序删除，避免索引错乱）
	Db_mu.Lock()
	defer Db_mu.Unlock()
	for i := len(toDelete) - 1; i >= 0; i-- {
		idx := toDelete[i]
		db_value_list = append(db_value_list[:idx], db_value_list[idx+1:]...)
	}

	return nil
}

// IO数据采集 订阅 接收
func Db_Subscriber(value Db_func) error {
	db_value_list = append(db_value_list, &value)
	return nil
}

/*
******************数据更新******************
 */

// 数据更新结构体
type Update_Value_type struct {
	Db_Value_type

	Last_Value any // 值
	Last_Time  time.Time
}

type Change_func func([]Update_Value_type) error

var (
	Update_Value_list []*Change_func
	Change_mu         sync.Mutex
)

// 数据更新 发布 发送
func Update_Publisher(value []Update_Value_type) (err error) {
	if len(value) == 0 {
		return nil
	}

	for _, v := range value {
		if !(v.Type == "bool" ||
			v.Type == "int" ||
			v.Type == "uint" ||

			v.Type == "int8" ||
			v.Type == "uint8" ||

			v.Type == "int16" ||
			v.Type == "uint16" ||

			v.Type == "int32" ||
			v.Type == "uint32" ||

			v.Type == "int64" ||
			v.Type == "uint64" ||

			v.Type == "float" ||
			v.Type == "float32" ||
			v.Type == "float64") {
			err := fmt.Errorf("ERROR 点位型不正确 %v", v.Type)
			return err
		}
		if v.Id == 0 {
			err = fmt.Errorf("ERROR 点位id不正确 %d", v.Id)
			return err
		}

		if v.Tag == "" {
			err = fmt.Errorf("ERROR 点位标识不正确 %d", v.Id)
			return err
		}

		var consistency bool
		consistency, err = Drive_Type_Map__Consistency(
			v.Tag,
			"",
			"",
			v.Type,
		)

		if err != nil {
			log.Print(err)
			err = nil
			continue
		}

		if !consistency {
			log.Printf("ERROR 接收到信息和配置不一致")
			continue
		}

	}

	var toDelete []int
	for i, v := range Update_Value_list {
		if v == nil {
			toDelete = append(toDelete, i)
			log.Printf("WARNING 关闭一个 IO采集 的订阅者 索引:%d", i)
			continue
		}
		err = (*v)(value)
		if err == Err_Publisher_Close {
			Update_Value_list[i] = nil
		} else if err != nil {
			log.Print(err)
		}

	}

	if len(toDelete) == 0 {
		return nil
	}

	// 统一删除需要关闭的订阅者（倒序删除，避免索引错乱）
	Change_mu.Lock()
	defer Change_mu.Unlock()
	for i := len(toDelete) - 1; i >= 0; i-- {
		idx := toDelete[i]
		db_value_list = append(db_value_list[:idx], db_value_list[idx+1:]...)
	}

	return nil
}

// 数据更新 订阅 接收
func Update_Subscriber(value Change_func) error {
	Update_Value_list = append(Update_Value_list, &value)
	return nil
}

func init() {
	Db_Subscriber(Update_Redis)
}

// 写人redis
func Update_Redis(db_value []Db_Value_type) (err error) {
	var (
		Last_Change_KeyValue map[string]string
		Update_KeyValue      []my_redis.KeyValue
		Key                  []string
		Update_Value         []Update_Value_type
	)
	for _, v := range db_value {
		Key = append(Key, fmt.Sprintf("point_value:%s", v.Tag))
	}

	Last_Change_KeyValue, err = my_redis.Read_Key_list(Key)
	if err != nil {
		return
	}

	for _, v := range db_value {
		var Last_Change Update_Value_type

		str, ok := Last_Change_KeyValue[fmt.Sprintf("point_value:%s", v.Tag)]
		if ok {
			Last_Change, err = Change_Json_Type(str)
			if err != nil {
				continue
			}
		}

		if Last_Change.Id == 0 || Last_Change.Tag == "" || Last_Change.Type == "" {
			Last_Change.Id = v.Id
			Last_Change.Tag = v.Tag
			Last_Change.Type = v.Type
		}

		if Last_Change.Type == "" {
			Last_Change.Type = v.Type
		}

		var w bool
		if Last_Change.Msg != v.Msg {
			Last_Change.Msg = v.Msg
			w = true
		}

		if Last_Change.Value != v.Value && v.Msg == "ok" {
			Last_Change.Last_Value = Last_Change.Value
			Last_Change.Last_Time = Last_Change.Time
			Last_Change.Value = v.Value
			Last_Change.Time = v.Time

			w = true
		}

		// 没有改变跳过
		if !w {
			continue
		}
		var jsonData []byte
		jsonData, err = json.Marshal(Last_Change)
		if err != nil {
			log.Printf("ERROR %v", err)
			return
		}

		Update_Value = append(Update_Value, Last_Change)

		Update_KeyValue = append(Update_KeyValue, my_redis.KeyValue{
			Key:   fmt.Sprintf("point_value:%s", v.Tag),
			Value: string(jsonData),
		})

	}
	Update_Value_len := len(Update_Value)
	Update_KeyValue_len := len(Update_KeyValue)

	if Update_Value_len != Update_KeyValue_len {
		err = fmt.Errorf("ERROR Update_Value和Update_KeyValue不一致  Update_Value_len:%d Update_KeyValue_len:%d", Update_Value_len, Update_KeyValue_len)
		return
	}

	if Update_Value_len == 0 {
		return
	}

	err = Update_Publisher(Update_Value)
	if err != nil {
		return
	}

	err = my_redis.Write_Key_list(Update_KeyValue)
	return
}

/*
******************点位查询对应设备id集合******************
 */
type Drive_Config_Type struct {
	Drive_Id   uint
	Drive_Type string
	RW_Cancel  string
	Value_Type string
}

type Register_Point_type struct {
	Drive_Config_Type
	Point_Tag string // 点位id
}

var (
	Drive_Type_Map       map[string]Drive_Config_Type // redis key值
	Drive_Type_Map_Mutex sync.RWMutex
)

// 注册点位
func Drive_Type_Map__Add(Points ...Register_Point_type) (err error) {
	if len(Points) == 0 {
		err = fmt.Errorf("ERROR 空数据")
		return
	}

	// c.data = make(map[uint]uint)
	Drive_Type_Map_Mutex.Lock()
	for _, v := range Points {
		Drive_Type_Map[v.Point_Tag] = v.Drive_Config_Type
	}
	Drive_Type_Map_Mutex.Unlock()
	return
}

// 注销点位
func Drive_Type_Map__Del(Points_Tag ...string) (err error) {
	if len(Points_Tag) == 0 {
		err = fmt.Errorf("ERROR 空数据")
		return
	}

	Drive_Type_Map_Mutex.Lock()
	for _, v := range Points_Tag {
		delete(Drive_Type_Map, v)
	}
	Drive_Type_Map_Mutex.Unlock()
	return
}

// 查询
func Drive_Type_Map__Query_PointTag(Points_Tag string) (Drive_Map Drive_Config_Type, err error) {
	var (
		ok bool
	)

	Drive_Type_Map_Mutex.RLock()
	Drive_Map, ok = Drive_Type_Map[Points_Tag]
	Drive_Type_Map_Mutex.RUnlock()

	if !ok {
		err = fmt.Errorf("ERROR 不存在的点位标识")
	}

	return
}

// 查询列表
func Drive_Type_Map__Query_PointId_list(Points_Tag ...string) (Drive_Map map[string]Drive_Config_Type, err error) {
	if len(Points_Tag) == 0 {
		err = fmt.Errorf("ERROR 空数据")
		return
	}

	var (
		ok bool
	)
	Drive_Map = make(map[string]Drive_Config_Type)

	Drive_Type_Map_Mutex.RLock()
	for _, v := range Points_Tag {
		var Drive_Config Drive_Config_Type
		Drive_Config, ok = Drive_Type_Map[v]
		if !ok {
			err = fmt.Errorf("ERROR 不存在的点位标识")
			continue
		}
		Drive_Map[v] = Drive_Config
	}
	Drive_Type_Map_Mutex.RUnlock()

	return
}

// 一致性
func Drive_Type_Map__Consistency(Points_Tag string, Drive_Type string, RW_Cancel string, Value_Type string) (consistency bool, err error) {

	Drive_Type_Map_Mutex.RLock()
	Drive_Map, ok := Drive_Type_Map[Points_Tag]
	Drive_Type_Map_Mutex.RUnlock()

	if !ok {
		err = fmt.Errorf("ERROR 不存在的点位标识")
	}

	if Drive_Map.Drive_Type != Drive_Type && Drive_Type != "" {
		err = fmt.Errorf("ERROR Drive_Type不一致 点位标识:%s 配置:%s, 传入:%s", Points_Tag, Drive_Map.Drive_Type, Drive_Type)
		return
	}

	if Drive_Map.RW_Cancel != RW_Cancel && RW_Cancel != "" {
		err = fmt.Errorf("ERROR RW_Cancel不一致 点位标识:%s 配置:%s, 传入:%s", Points_Tag, Drive_Map.RW_Cancel, RW_Cancel)
		return
	}

	if Drive_Map.Value_Type != Value_Type && Value_Type != "" {
		err = fmt.Errorf("ERROR Value_Type不一致 点位标识:%s 配置:%s, 传入:%s", Points_Tag, Drive_Map.Value_Type, Value_Type)
		return
	}

	consistency = true
	return
}

// 点位查询对应设备id集合 初始化
// func Drive_Type_Map_Init() (err error) {
// 	Drive_Type_Map = &Drive_Type_Map_struct{
// 		data: make(map[string]Drive_Config_Type), // 初始化map（核心）
// 	}

// 	var drive_configs []db_mysql.Drive_Config_Db_type
// 	drive_configs, err = db_mysql.Drive_Config__Query("", 0, 0)
// 	if err != nil {
// 		log.Print(err.Error())
// 		return
// 	}

// 	var points []Register_Point_type
// 	for _, drive_config := range drive_configs {
// 		var point_configs []db_mysql.Points_Config_Db_type
// 		point_configs, err = db_mysql.Points_Config__Query(drive_config.Field.Id, 0, 0)
// 		if err != nil {
// 			log.Print(err.Error())
// 			continue
// 		}

// 		for _, point_config := range point_configs {

// 			if drive_config.Field.Id != point_config.Field.Drive_Id {
// 				err = fmt.Errorf("ERROR 设备id和 点位中的设备id不一致 设备id:%d 点位中的设备id:%d", drive_config.Field.Id, point_config.Field.Drive_Id)
// 				continue
// 			}

// 			points = append(points, Register_Point_type{
// 				Point_Tag: point_config.Field.Tag,
// 				Drive_Config_Type: Drive_Config_Type{
// 					Drive_Id:   drive_config.Field.Id,
// 					Drive_Type: drive_config.Field.Type,
// 					RW_Cancel:  point_config.Field.RW_Cancel,
// 					Value_Type: point_config.Field.Value_Type,
// 				},
// 			})

// 		}
// 	}

// 	err = Drive_Type_Map.Add(points...)
// 	if err != nil {
// 		log.Print(err.Error())
// 		return
// 	}
// 	return
// }

// 包入口 这个不要多线程运行,需要曾提前运行
func New() (err error) {
	// err = Drive_Type_Map_Init()

	Drive_Type_Map = make(map[string]Drive_Config_Type)
	return err
}
