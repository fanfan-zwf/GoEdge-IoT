package mqtt_rpc

import (
	"fmt"
	"log"
	"main/Init"
	"time"
)

func App_Restart(req []byte) (rep []byte, err error) {
	type Req struct {
	}

	return jsonWrap(req, func(r Req) (rep string, err error) {
		log.Println("未开发")
		rep = "未开发"
		err = fmt.Errorf("测试错误")
		return
	})
}

func register() {
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/Order/App/Restart", App_Restart)
}

type Collector__Carry_type struct {
	Id   uint   // 采集器标识
	Name string // 采集器名称
	Uuid string // 采集器uuid
}

type Collector_Info_type struct {
	Id                 uint      // 采集 Id
	Label              string    // 标识
	Creation_Time      time.Time // 创建时间
	Uuid               string    // Uuid
	Sn                 string    // 设备 sn
	User_Id            uint      // 创建用户 id
	Version            string    // 版本
	Last_Activity_Time time.Time // 最后活动时间
	Equipment_Id       uint      // 设备 id
	Name               string    // 设备名称
}

func Collector_Info__Count(page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		page     uint
		pageSize uint
	}
	err = jsonCall(Req{page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Collector/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

func Collector_Info__Query(page uint, pageSize uint) (resp []Collector_Info_type, err error) {
	type Req struct {
		page     uint
		pageSize uint
	}

	err = jsonCall(Req{page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Collector/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

type Drive__Carry_type struct {
	Id   uint   // 驱动id唯一标识符
	Type string // 驱动类型
	Name string // 驱动名称
}

type Drive_Config_type struct {
	Collector Collector__Carry_type

	Id            uint      // 驱动id
	Name          string    // 驱动名称
	Config        string    // json配置参数
	Type          string    // 驱动类型
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
}

func Drive_Config__Count(collectorId uint, driveType string, page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		collectorId uint
		driveType   string
		page        uint
		pageSize    uint
	}
	err = jsonCall(Req{collectorId: collectorId, driveType: driveType, page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

func Drive_Config__Query(collectorId uint, driveType string, page uint, pageSize uint) (resp []Collector_Info_type, err error) {
	type Req struct {
		collectorId uint
		driveType   string
		page        uint
		pageSize    uint
	}

	err = jsonCall(Req{collectorId: collectorId, driveType: driveType, page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

// 点位配置更新结构体
type Points_Config_Update_type struct {
	Id          uint   // 点位id
	Tag         string // 点位标识
	Description string // 点位描述
	RW_Cancel   string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type  string // 输出类型
	Config      string
}

// 点位配置结构体
type Points_Config_type struct {
	Collector Collector__Carry_type
	Drive     Drive__Carry_type

	Points_Config_Update_type
	Creation_Time time.Time // 创建时间

}

func Points_Config__Count(driveid uint, page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		driveid  uint
		page     uint
		pageSize uint
	}
	err = jsonCall(Req{driveid: driveid, page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR ：%v", err)
		log.Print(err)
	}
	return
}

func Points_Config__Query(driveid uint, page uint, pageSize uint) (resp []Points_Config_type, err error) {
	type Req struct {
		driveid  uint
		page     uint
		pageSize uint
	}

	err = jsonCall(Req{driveid: driveid, page: page, pageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR %v", err)
		log.Print(err)
	}
	return
}
