package mqtt_rpc

import (
	"log"
	"main/Init"
	"main/cloud"
	m_mysql "main/db/mysql"
	"time"

	"encoding/json"
	"fmt"
)

// jsonWrap RPC 通用包装：自动处理 JSON 反序列化/序列化
// T: 请求结构体类型  R: 响应结构体类型
func jsonWrap[T any, R any](req []byte, business func(req T) (R, error)) (rep []byte, err error) {

	// 数据解密
	var decryption []byte
	decryption, err = cloud.Receive__CRC32_Aes_Gzip(req, Init.Config.APP.AesPasswd)
	if err != nil {
		log.Printf("ERROR 解密或解压失败 %s", err)
		return nil, err
	}

	// 1. 自动反序列化 JSON → 请求结构体
	var reqData T
	if err = json.Unmarshal(decryption, &reqData); err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return nil, err
	}

	// 2. 执行业务逻辑（你只需要写这里）
	respData, err := business(reqData)
	if err != nil {
		log.Println("ERROR 业务执行失败：", err)
		return nil, err
	}

	// 3. 自动序列化 结构体 → JSON
	respBytes, err := json.Marshal(respData)
	if err != nil {
		log.Println("ERROR JSON转换失败：", err)
		return nil, err
	}

	// 数据加密
	var encryption []byte
	encryption, err = cloud.Send__CRC32_Aes_Gzip(respBytes, Init.Config.APP.AesPasswd)
	if err != nil {
		log.Println("ERROR 加密失败：", err)
		return nil, err
	}

	return encryption, nil
}

// rpcCall 客户端RPC调用包装：自动 JSON + 加解密 + 发送 + 解析
func jsonCall[Req any, Resp any](
	reqData Req,
	respData *Resp,
	broker string,
	topic string,
	method string,
	timeout time.Duration,
) error {
	// 1. 序列化请求
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR 请求打包失败：", err)
		return err
	}

	// 2. 加密
	encBytes, err := cloud.Send__CRC32_Aes_Gzip(reqBytes, Init.Config.APP.AesPasswd)
	if err != nil {
		log.Println("ERROR 请求加密失败：", err)
		return err
	}

	// 3. 调用 MQTT RPC
	respBytes, err := M.Call(broker, topic, method, encBytes, timeout)
	if err != nil {
		log.Println("ERROR RPC调用失败：", err)
		return err
	}

	// 4. 解密
	decBytes, err := cloud.Receive__CRC32_Aes_Gzip(respBytes, Init.Config.APP.AesPasswd)
	if err != nil {
		log.Println("ERROR 响应解密失败：", err)
		return err
	}

	// 5. 反序列化到响应结构体
	if err := json.Unmarshal(decBytes, respData); err != nil {
		log.Println("ERROR 响应解析失败：", err)
		return err
	}

	return nil
}
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

	M.Register(Init.Config.Mqtt_Rpc.Broker, "/V1.0/App/HeartBeat", App_HeartBeat)
}

// 重启采集服务软件
func App_Restart(uuid string) (err error) {
	type Req struct {
		Uuid string
	}
	type Resp struct {
		Status string
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
		3*time.Second,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		return
	}

	if resp != "OK" {
		err = fmt.Errorf("ERROR 响应错误：%s", resp)
		return
	}
	return
}
