/*
* 日期: 2026.11.13
* 作者: 范范zwf
* 作用: 内部 MQTT 功能。使用一条专用 MQTT 连接（由 mqtt.New 根据
*       Internal_Mqtt 的 broker/账号等参数建立），
*       以 "主题前缀 /IOT/Collector/{label}"（前缀可在 config 自定义）为根，对外提供采集器的
*       读写、运维与监控主题：
*
*         订阅（下行，统一以 base+"/#" 通配订阅，再按后缀分发）：
*           /write         写数据：数组 json，以全局唯一 tag 为键，带类型/值（可选时间戳）
*           /restart       重启：json，target 区分 driver / service
*           /token         增加临时 handler token：json，长/短 token，直写 Redis
*           /config/update 更新所有配置：json，携带 http 下载地址（表格 xlsx），下载后全量导入
*
*         发布（上行）：
*           /data             读数据：以 tag 为键，点位值变化即时推送 + 每 data_full_interval 全量推送一次
*           /alarm            报警事件：数组 json，每元素含报警配置全部字段 + 报警值 + 报警时间，触发/恢复即推
*           /status/<功能>     当前监控状态：sysinfo 每个功能一个子主题（如 /status/cpu/percent、/status/mem）
*           /result           操作结果回执（重启 / 令牌 / 配置更新）
*
*       本包使用的连接 client_id 已在 config 层默认取 APP.label；本包只负责主题拼装与收发调度。
 */
package internalmqtt

import (
	"collector/internal/config"
	"collector/internal/infra/mqtt"
	"fmt"
	"log"
	"strings"
	"sync"
)

// baseTopic 内部功能主题的公共前缀：topicPrefix + "/" + label（如 /IOT/Collector/<label>）。
var baseTopic string

// stopChan 由 Stop 关闭，用于退掉周期发布协程；stopOnce 保证重复调用不会 panic。
var (
	stopChan = make(chan struct{})
	stopOnce sync.Once
)

// Stop 停止内部 MQTT 的周期发布协程，供优雅退出调用（应在关闭 MQTT 连接之前执行）。
func Stop() {
	stopOnce.Do(func() { close(stopChan) })
}

// cfg 内部 MQTT 生效配置（含默认值），Start 时从全局配置拷贝一份。
var cfg internalConfig

// internalConfig 运行时使用的配置快照。
type internalConfig struct {
	enable          bool
	broker          string
	subWrite        string
	subRestart      string
	subToken        string
	subConfigUpdate string
	pubData         string
	pubAlarm        string
	pubStatusPrefix string
	pubResult       string
}

// Start 初始化内部 MQTT 功能：校验开关与 broker，拼装主题前缀，注册订阅并启动发布协程。
// 传入：无（读取全局 config）
// 返回：错误（broker 连接不存在等）；未启用时返回 nil 且不报错
func Start() error {
	if !config.Config.Internal_Mqtt.Enable {
		log.Printf("INFO 内部MQTT功能已禁用")
		return nil
	}

	label := config.Config.APP.Label
	if label == "" {
		return fmt.Errorf("内部MQTT启动失败：APP.label 未配置，无法拼装主题前缀")
	}

	// 内部功能使用 mqtt 包在 New() 中建立的独立连接
	broker := mqtt.InternalBrokerName
	if !mqtt.GlobalMQTT.Exist(broker) {
		return fmt.Errorf("内部MQTT启动失败：独立连接[%s]不存在或未初始化（检查 Internal_Mqtt.broker 地址）", broker)
	}

	cfg = internalConfig{
		enable:          true,
		broker:          broker,
		subWrite:        config.Config.Internal_Mqtt.SubWrite,
		subRestart:      config.Config.Internal_Mqtt.SubRestart,
		subToken:        config.Config.Internal_Mqtt.SubToken,
		subConfigUpdate: config.Config.Internal_Mqtt.SubConfigUpdate,
		pubData:         config.Config.Internal_Mqtt.PubData,
		pubAlarm:        config.Config.Internal_Mqtt.PubAlarm,
		pubStatusPrefix: config.Config.Internal_Mqtt.PubStatusPrefix,
		pubResult:       config.Config.Internal_Mqtt.PubResult,
	}

	baseTopic = joinTopic(config.Config.Internal_Mqtt.TopicPrefix, label)
	log.Printf("INFO 内部MQTT功能启动，独立连接=%s 主题前缀=%s", broker, baseTopic)

	// 订阅（下行）：以 base+"/#" 通配订阅一个主题，收到后按后缀分发
	if err := subscribeCommands(); err != nil {
		return err
	}

	// 发布（上行）：数据推送 + 报警事件推送 + 监控状态推送
	startDataPublisher()
	startAlarmPublisher()
	startStatusPublisher()

	return nil
}

// joinTopic 安全拼接主题片段，避免出现重复或前后多余的 "/"。
func joinTopic(parts ...string) string {
	elems := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			elems = append(elems, p)
		}
	}
	return "/" + strings.Join(elems, "/")
}

// publish 向指定后缀发布一条消息（QoS 由 mqtt 包统一处理）。
func publish(suffix string, data []byte) error {
	topic := joinTopic(baseTopic, suffix)
	if err := mqtt.Send(cfg.broker, topic, data); err != nil {
		log.Printf("ERROR 内部MQTT发布失败 topic=%s: %v", topic, err)
		return err
	}
	return nil
}

// resultSuffix 拼出某操作的结果回执主题（用于异步动作的反馈，如配置更新）。
func resultSuffix(action string) string {
	return joinTopic(cfg.pubResult, action)
}
