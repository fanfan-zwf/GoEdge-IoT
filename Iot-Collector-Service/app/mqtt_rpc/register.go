package mqtt_rpc

import (
	"fmt"
	"log"
	"main/Init"
	"main/cloud"
	"time"

	"encoding/json"
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
func App_Restart(req []byte) (rep []byte, err error) {
	type Req struct {
	}

	return jsonWrap(req, func(r Req) (rep string, err error) {
		rep = "未开发"
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
		5*time.Second,
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
		5*time.Second,
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
		5*time.Second,
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
		5*time.Second,
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
		5*time.Second,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
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
		5*time.Second,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}
