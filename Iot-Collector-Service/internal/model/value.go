package model

import (
	"errors"
	"collector/internal/config"
	"collector/internal/repo/mysql"
	"strings"
	"time"
)


type Value_type struct {
	PointId uint      // 点位id
	Tag     string    // 标签
	Value   any       // 点位值
	Type    int       // 输出类型（类型编码，参见 conv convertType.go）
	Msg     string    // 状态信息
	Time    time.Time // 读取时间
}

// FullConfig 驱动全配置（驱动配置 + 该驱动下的所有点位配置）
type FullConfig_type struct {
	Drive  mysql.Drive_Config_Query_type
	Points []mysql.Point_Config_Query_type
}

// TagWithDeviceId 为点位tag添加设备id前缀
// tag格式: //设备id//标识符（长度可变）
// tag必须以//或/开头，否则返回错误
// 如果tag已包含设备id（以//开头），验证后返回
// 如果tag不包含设备id（以/开头），在前面增加 //设备id//
func TagWithDeviceId(tag string) (string, error) {
	deviceId := config.Config.APP.Label
	if deviceId == "" {
		return "", errors.New("设备标识符(Label)未配置")
	}

	if tag == "" {
		return "", errors.New("tag不能为空")
	}

	// tag必须以//或/开头
	if !strings.HasPrefix(tag, "//") && !strings.HasPrefix(tag, "/") {
		return "", errors.New("tag格式错误，必须以//或/开头")
	}

	// 如果tag不以//开头，去掉开头的/后拼接设备id
	if !strings.HasPrefix(tag, "//") {
		tag = "//" + deviceId + "//" + strings.TrimLeft(tag, "/")
	}

	// 验证是否恰好2组//
	if strings.Count(tag, "//") != 2 {
		return "", errors.New("tag格式错误，必须恰好包含2组//")
	}

	return tag, nil
}

// TagSplitDeviceId 从tag中拆分出设备id
// tag格式: //设备id//标识符（整个字符串只能有2组//）
// 返回: 设备id, 是否包含设备id
func TagSplitDeviceId(tag string) (string, bool) {
	// 不以//开头，说明不包含设备id
	if !strings.HasPrefix(tag, "//") {
		return "", false
	}

	// 统计 // 的数量，必须恰好是2组
	count := strings.Count(tag, "//")
	if count != 2 {
		return "", false
	}

	// 从第3个字符开始找设备id结束位置
	rest := tag[2:] // 去掉开头的 //

	// 查找 // 的位置
	idx := strings.Index(rest, "//")
	if idx == -1 {
		return "", false
	}

	// 返回设备id部分
	return rest[:idx], true
}
