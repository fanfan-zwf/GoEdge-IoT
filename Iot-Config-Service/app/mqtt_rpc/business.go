package mqtt_rpc

import (
	"main/Init"
	m_mysql "main/db/mysql"
	"time"

	"fmt"
)

func Collector_Info__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		Page     uint
		PageSize uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Collector_Info__Count(r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Collector_Info__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		Page     uint
		PageSize uint
	}

	type Resp []m_mysql.Collector_Info_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Collector_Info__Query(r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Collector_Info__Search_Field(req []byte) (rep []byte, err error) {
	type Req struct {
		Field    string
		Quantity uint
		Vague    string
	}

	type Resp []m_mysql.Collector_Info_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Collector_Info__Search_Field(r.Field, r.Quantity, r.Vague)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败123：%v", err)
		}
		return
	})
}
func Drive_Config__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		CollectorId []uint
		DriveType   []string
		Page        uint
		PageSize    uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Drive_Config__Count(r.CollectorId, r.DriveType, r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Drive_Config__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		CollectorId []uint
		DriveType   []string
		Page        uint
		PageSize    uint
	}

	type Resp []m_mysql.Drive_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Drive_Config__Query(r.CollectorId, r.DriveType, r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Points_Config__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		Collector_Id []uint
		Driveid      []uint
		Page         uint
		PageSize     uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Points_Config__Count(r.Collector_Id, r.Driveid, r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Points_Config__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		Collector_Id []uint
		Driveid      []uint
		Page         uint
		PageSize     uint
	}

	type Resp []m_mysql.Points_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Points_Config__Query(r.Collector_Id, r.Driveid, r.Page, r.PageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func App_HeartBeat(req []byte) (rep []byte, err error) {
	type Req struct {
		Uuid      string
		Heartbeat time.Time
	}

	type Resp []m_mysql.Points_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		err = m_mysql.Collector_Info__Last_Activity_Time(r.Uuid, r.Heartbeat)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func register() {
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Collector/Count", Collector_Info__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Collector/Query", Collector_Info__Query)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Collector/Search/Field", Collector_Info__Search_Field)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Drive/Count", Drive_Config__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Drive/Query", Drive_Config__Query)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Points/Count", Points_Config__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Points/Query", Points_Config__Query)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/App/HeartBeat", App_HeartBeat) // 心跳
}

// 重启采集服务软件
func App_Restart(uuid string) (err error) {
	if uuid == "" {
		err = fmt.Errorf("ERROR uuid参数错误")
		return
	}

	type Req struct {
		Uuid string
	}

	var listen_topic string
	listen_topic, err = m_mysql.Collector_Info__Query_Uuid__ListenTopic(uuid)
	if err != nil {
		err = fmt.Errorf("ERROR 查询失败：%v", err)
		return
	}

	var resp string
	err = jsonCall(Req{Uuid: uuid}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		listen_topic,
		"/Order/App/Restart",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR ：%v", err)
		return
	}

	if resp != "ok" {
		err = fmt.Errorf("ERROR 响应错误： %+v ", resp)
		return
	}
	return
}

// 采集服务同步
func Collector_Synchronise_Config(uuid string) (err error) {
	if uuid == "" {
		err = fmt.Errorf("ERROR uuid参数错误")
		return
	}

	type Req struct {
		Uuid string
	}

	var listen_topic string
	listen_topic, err = m_mysql.Collector_Info__Query_Uuid__ListenTopic(uuid)
	if err != nil {
		err = fmt.Errorf("ERROR 查询失败：%v", err)
		return
	}

	var resp string
	err = jsonCall(Req{Uuid: uuid}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		listen_topic,
		"/Order/Collector/Config",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR ：%v", err)
		return
	}

	if resp != "ok" {
		err = fmt.Errorf("ERROR 响应错误： %+v ", resp)
		return
	}
	return
}

// 采集服务重载驱动
func Collector_Reload(uuid string) (err error) {
	if uuid == "" {
		err = fmt.Errorf("ERROR uuid参数错误")
		return
	}

	type Req struct {
	}

	var listen_topic string
	listen_topic, err = m_mysql.Collector_Info__Query_Uuid__ListenTopic(uuid)
	if err != nil {
		err = fmt.Errorf("ERROR 查询失败：%v", err)
		return
	}

	var resp string
	err = jsonCall(Req{}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		listen_topic,
		"/Order/Collector/Reload",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR ：%v", err)
		return
	}

	if resp != "ok" {
		err = fmt.Errorf("ERROR 响应错误： %+v ", resp)
		return
	}
	return
}
