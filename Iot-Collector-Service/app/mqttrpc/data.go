package mqttrpc

import (
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/app/mqttbase"
	"main/db/db_point"

	"encoding/json"
	"log"
)

// 点更新值
func Point_Push_Value(data []fullConfig.Value_type) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return mqttbase.Send(Init.Config.Mqtt_Rpc.Example, Init.Config.Mqtt_Rpc.Point_Push_Value, jsonBytes)
}

// 点下发值
func Point_Down_value(callback func([]fullConfig.Value_type)) error {
	return mqttbase.Subscribe(Init.Config.Mqtt_Rpc.Example, Init.Config.Mqtt_Rpc.Point_Down_value, func(broker string, topic string, data []byte) {
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
	return mqttbase.Send(Init.Config.Mqtt_Rpc.Example, Init.Config.Mqtt_Rpc.Point_Alarm_Value, jsonBytes)
}

func init() {
	if !Init.Config.Mqtt_Rpc.Enable {
		return
	}
	db_point.Update_Subscriber(Point_Push_Value)
	db_point.Alarm_Subscriber(Point_Alarm_Value)
}
