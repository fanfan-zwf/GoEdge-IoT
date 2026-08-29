package fullConfig

import (
	"main/db/mysql"
	"time"
)

type Value_type struct {
	DeviceId string    // 设备id
	PointId  uint      // 点位id
	Value    any       // 点位值
	Type     string    // 输出类型
	Msg      string    // 状态信息
	Time     time.Time // 读取时间
}

// FullConfig 驱动全配置（驱动配置 + 该驱动下的所有点位配置）
type FullConfig_type struct {
	Drive  mysql.CollectorGet_Drive_Config_type
	Points []mysql.CollectorGet_Point_Config_type
}
