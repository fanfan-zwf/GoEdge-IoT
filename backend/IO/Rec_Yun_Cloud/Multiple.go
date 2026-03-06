/*
* 日期: 2025.5.13 PM17:26
* 作者: 范范zwf
* 作用: 开始
 */

package Rec_Yun_Cloud

import (
	"encoding/json"
	"fmt"
	"log"
	"main/db/db_point"
	my_mysql "main/db/mysql"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	package_drive_type string = "Rec_Yun_Cloud"
)

var Not_exist error = fmt.Errorf("Not exist")

/*
******************实例状态执行******************
 */

var (
	Status_Execution_Mu *sync.Mutex
)

/*
******************实例存储******************
 */

var (
	Drive_Instance_list_riveMu sync.RWMutex
	Drive_Instance_list        map[uint]*Connect_struct
)

func init() {
	Drive_Instance_list = make(map[uint]*Connect_struct)
}

// 读取实例
func Example_Read(id uint) (connect *Connect_struct, err error) {
	Drive_Instance_list_riveMu.RLock()
	defer Drive_Instance_list_riveMu.RUnlock()

	var ok bool
	connect, ok = Drive_Instance_list[id]
	if !ok {
		err = Not_exist
	}

	return
}

// 增加实例
func Example_Write(id uint, connect *Connect_struct) (err error) {
	Drive_Instance_list_riveMu.Lock()
	defer Drive_Instance_list_riveMu.Unlock()

	Drive_Instance_list[id] = connect

	return
}

/*
******************实例状态******************
 */
func Start_collecting() {

}

// 运行实例
func Example_Run(id uint) (err error) {
	if id == 0 {
		err = fmt.Errorf("ERROR id不应该是0")
		return
	}

	var connect *Connect_struct
	connect, err = Example_Read(id)
	if err == Not_exist {
		connect = &Connect_struct{}
	} else if err != nil {
		fmt.Print(id, "err != nil && err != Not_exist   \n")
		return
	}

	if connect.Client != nil {
		err = fmt.Errorf("ERROR 驱动已经运行 设备id:%d", connect.Drive.Id)
		return
	}

	connect.Drive, err = my_mysql.Drive_Config__Query_DriveId(id)
	if connect.Drive.Id != id {
		err = fmt.Errorf("ERROR 不是指定的设备id 读取设备id%d 需要设备id%d", connect.Drive.Id, id)
		return
	}
	log.Print("INFO", connect.Drive)

	err = json.Unmarshal([]byte(connect.Drive.Config), &connect.Connect_Config)
	if err != nil {
		err = fmt.Errorf("ERROR %v", err)
		log.Print(err)
		return
	}

	if connect.Drive.Type != package_drive_type {
		err = fmt.Errorf("ERROR 读取设备不是设定的类型 设备id:%d 读取类型%s 需要类型%s", id, connect.Drive.Type, package_drive_type)
		return
	}

	var points_configs []my_mysql.Points_Config_type
	points_configs, err = my_mysql.Points_Config__Query(id, 0, 0)
	connect.Points = points_configs

	var Register_Points []db_point.Register_Point_type //注册点位
	for _, v := range points_configs {
		Register_Points = append(Register_Points, db_point.Register_Point_type{
			Point_Tag: v.Tag,
			Drive_Config_Type: db_point.Drive_Config_Type{
				Drive_Id:   v.Drive_Id,
				Drive_Type: package_drive_type,
				RW_Cancel:  v.RW_Cancel,
				Value_Type: v.Value_Type,
			},
		})
	}
	err = db_point.Drive_Type_Map__Add(Register_Points...)
	if err != nil {
		var tags []string
		for _, v := range Register_Points {
			tags = append(tags, v.Point_Tag)
		}
		db_point.Drive_Type_Map__Del(tags...)
	}

	err = Example_Write(id, connect)
	if err != nil {
		return
	}

	// 运行驱动
	log.Printf("INFO MQTTCommand_Publish_Init")
	err = connect.MQTTCommand_Publish_Init()
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("INFO Packet")
	err = connect.Packet()
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("INFO Connect")
	err = connect.Connect()
	if err != nil {
		log.Print(err)
		return
	}

	return
}

// 停止实例
func Example_Stop(id uint) (err error) {
	if id == 0 {
		err = fmt.Errorf("ERROR id不应该是0")
		return
	}

	var connect *Connect_struct
	connect, err = Example_Read(id)
	if err != nil {
		return
	}

	var tags []string
	for _, v := range connect.Points {
		tags = append(tags, v.Tag)
	}

	err = db_point.Drive_Type_Map__Del(tags...)
	if err != nil {
		return
	}

	err = connect.Close()
	time.Sleep(1 * time.Second)

	return
}

// 重启实例
func Example_Restart(id uint) (err error) {
	if id == 0 {
		err = fmt.Errorf("ERROR id不应该是0")
		return
	}

	err = Example_Stop(id)
	if err != nil {
		return
	}

	time.Sleep(1 * time.Second)
	err = Example_Run(id)
	return
}

func New() (err error) {
	log.Printf("INFO 开始运行mqtt服务")
	var drive_configs []my_mysql.Drive_Config_type
	drive_configs, err = my_mysql.Drive_Config__Query_DriveType(package_drive_type)
	if err != nil {
		return
	}

	for _, drive_config := range drive_configs {
		log.Print("INFO drive_ids", drive_config)
		err = Example_Run(drive_config.Id)
		if err != nil {
			log.Print(err.Error())
			err = nil
		}
	}

	return
}
