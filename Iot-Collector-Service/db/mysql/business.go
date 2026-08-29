/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"time"
)

/*
***************采集配置结构体***************
 */

type Collector__Carry_type struct {
	Id   uint   // 采集器标识
	Name string // 采集器名称
	Uuid string // 采集器uuid
}

// 采集配置增加结构体
type Collector_Config_Add_type struct {
	Label   string // 标识
	Uuid    string // Uuid
	Name    string // 设备名称
	User_Id uint   // 用户id
}

type Collector_Config_Update_type struct {
	Id   uint   // 采集 Id
	Name string // 设备名称
}

// 采集配置结构体
type Collector_Config_type struct {
	Id                 uint      // 采集 Id
	Label              string    // 标识
	Creation_Time      time.Time // 创建时间
	Uuid               string    // Uuid (修正为 string)
	Sn                 string    // 设备 sn
	User_Id            uint      // 创建用户 id
	Version            string    // 版本
	Last_Activity_Time time.Time // 最后活动时间
	Equipment_Id       uint      // 设备 id
	Name               string    // 设备名称
}

/*
***************驱动配置结构体***************
 */

type CollectorGet_Drive_Config_type struct {
	Id            uint   // 递增id
	Collector_Id  uint   // 采集id
	Type          string // 驱动类型
	Config        string // json配置参数
	Points_Length uint   // 点位数量
}

/*
***************点位配置结构体***************
 */
type CollectorGet_Point_Config_type struct {
	Id         uint   // 点位id
	Drive_Id   uint   // 驱动id
	Config     string // 配置信息
	RW_Cancel  int    // 读写方式读写方式 1：禁止； 2：只读； 3：只写； 4：读写；
	Value_Type int    // 输出类型 1：bool； 2：int8； 3：uint8； 4：int16； 5：uint16； 6：int32； 7：uint32； 8：int64； 9：uint64； 10：int； 11：uint； 12：float32； 13：float64； 14：float； 15：string；
}

/*
***************报警***************
 */

type CollectorGet_Alarm_Config_type struct {
	Id       uint   // 报警id
	Point_Id uint   // 点位id
	Config   string // 报警配置
	Group    int    // 报警组
}

/*
***************历史配置***************
 */

type CollectorGet_History_Config_type struct {
	Id       uint   // 历史id
	Point_Id uint   // 点位id
	Config   string // 历史配置
}
