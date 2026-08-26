package mqttcommand

import (
	"fmt"
	"main/app/mqttbase"
	"main/cloud"
	"main/db/mysql"
)

// 下发命令
func Collector_Command(Uuid string, c string) error {
	res, err := mysql.Collector_Config__Query_MqttByLabel(Uuid)
	if err != nil {
		return err
	}
	if res.Id == 0 {
		return fmt.Errorf("ERROR 查询采集配置失败，Label不存在")
	}
	if res.Mqtt_Topic == "" {
		return fmt.Errorf("ERROR 查询采集配置失败，Mqtt_Topic为空")
	}

	example, exist := cloud.GetKVValue(res.Mqtt_Topic, "Mqtt_Example")
	if !exist {
		return fmt.Errorf("ERROR 查询采集配置失败，Mqtt_example为空")
	}

	topic, exist := cloud.GetKVValue(res.Mqtt_Topic, "Mqtt_Topic")
	if !exist {
		return fmt.Errorf("ERROR 查询采集配置失败，Mqtt_Topic为空")
	}

	return mqttbase.Send(example, topic, []byte(c))
}
