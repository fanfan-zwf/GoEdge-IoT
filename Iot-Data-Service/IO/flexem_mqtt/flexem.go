package flexemmqtt

import (
	"main/app/mqttbase"
)

func InitGlobal() {
	mqttbase.NewManager()
}
