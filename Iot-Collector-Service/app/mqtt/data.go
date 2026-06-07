package mqtt

import (
	"log"
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/db_point"

	"encoding/json"
)

// 点更新值
func Point_Push_Value(data []fullConfig.Value_type) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return Send(Init.Config.Mqtt.Broker, Init.Config.Mqtt.Point_Push_Value, jsonBytes)
}

// 点下发值
func Point_Down_value(callback func([]fullConfig.Value_type)) error {
	return Subscribe(Init.Config.Mqtt.Broker, Init.Config.Mqtt.Point_Down_value, func(data []byte) {
		var down []fullConfig.Value_type
		err := json.Unmarshal(data, &down)
		if err != nil {
			log.Println("ERROR 解析JSON失败:", err)
			return
		}
		callback(down)
	})
}

// 点更新值
func Point_Alarm_Value(data []db_point.Alarm_type) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return Send(Init.Config.Mqtt.Broker, Init.Config.Mqtt.Point_Alarm_Value, jsonBytes)
}

func init() {
	if !Init.Config.Mqtt.Enable {
		return
	}
	db_point.Update_Subscriber(Point_Push_Value)
	db_point.Alarm_Subscriber(Point_Alarm_Value)
}
