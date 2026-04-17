package mqtt_rpc

import (
	"main/Init"
	m_mysql "main/db/mysql"
	"time"

	"fmt"
)

func Collector_Info__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		page     uint
		pageSize uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Collector_Info__Count(r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Collector_Info__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		page     uint
		pageSize uint
	}

	type Resp []m_mysql.Collector_Info_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Collector_Info__Query(r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Drive_Config__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		collectorId uint
		driveType   string
		page        uint
		pageSize    uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Drive_Config__Count(r.collectorId, r.driveType, r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Drive_Config__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		collectorId uint
		driveType   string
		page        uint
		pageSize    uint
	}

	type Resp []m_mysql.Drive_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Drive_Config__Query(r.collectorId, r.driveType, r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Points_Config__Count(req []byte) (rep []byte, err error) {
	type Req struct {
		driveid  uint
		page     uint
		pageSize uint
	}

	return jsonWrap(req, func(r Req) (rep uint, err error) {
		rep, err = m_mysql.Points_Config__Count(r.driveid, r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func Points_Config__Query(req []byte) (rep []byte, err error) {
	type Req struct {
		driveid  uint
		page     uint
		pageSize uint
	}

	type Resp []m_mysql.Points_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		rep, err = m_mysql.Points_Config__Query(r.driveid, r.page, r.pageSize)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func App_HeartBeat(req []byte) (rep []byte, err error) {
	type Req struct {
		Uuid      string
		heartbeat time.Time
	}

	type Resp []m_mysql.Points_Config_type
	return jsonWrap(req, func(r Req) (rep Resp, err error) {
		err = m_mysql.Collector_Info__Last_Activity_Time(r.Uuid, r.heartbeat)
		if err != nil {
			err = fmt.Errorf("ERROR 查询失败：%v", err)
		}
		return
	})
}

func register() {
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Collector/Count", Collector_Info__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Collector/Query", Collector_Info__Query)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Drive/Count", Drive_Config__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Drive/Query", Drive_Config__Query)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Points/Count", Points_Config__Count)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/Get/Points/Query", Points_Config__Query)

	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/App/HeartBeat", App_HeartBeat) // 心跳
}

// 重启采集服务软件
func App_Restart(uuid string) (err error) {
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

	if resp != "OK" {
		err = fmt.Errorf("ERROR 响应错误： %+v ", resp)
		return
	}
	return
}
