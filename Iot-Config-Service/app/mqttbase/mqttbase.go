package mqttbase

import (
	"fmt"
	"log"
	"main/Init"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttConfig 纯MQTT连接配置，无RPC相关字段
type MqttConfig struct {
	Broker            string
	Username          string
	Password          string
	ClientID          string
	SetCleanSession   bool
	SetAutoReconnect  bool
	SetConnectTimeout time.Duration
	SetWriteTimeout   time.Duration
	SetKeepAlive      time.Duration
}

// subscription 记录订阅信息
type subscription struct {
	Topic    string
	QoS      byte
	Callback func(broker string, topic string, data []byte)
}

type mqttInstance struct {
	client        mqtt.Client
	subscriptions map[string]*subscription // key: topic
	subMu         sync.RWMutex
}

// Manager MQTT连接管理器
type Manager struct {
	inst map[string]*mqttInstance
	mu   sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		inst: make(map[string]*mqttInstance),
	}
}

// Add 添加单个MQTT连接
func (m *Manager) Add(name string, cfg MqttConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(cfg.SetCleanSession)
	opts.SetAutoReconnect(cfg.SetAutoReconnect)
	opts.SetConnectTimeout(cfg.SetConnectTimeout)
	opts.SetWriteTimeout(cfg.SetWriteTimeout)
	opts.SetKeepAlive(cfg.SetKeepAlive)

	// 创建实例并保存引用，以便在回调中使用
	instance := &mqttInstance{
		subscriptions: make(map[string]*subscription),
	}

	// 设置连接丢失处理器
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("MQTT[%s] 连接断开: %v", name, err)
	})

	// 设置重连处理器
	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		log.Printf("MQTT[%s] 尝试重连...", name)
	})

	// 设置连接成功处理器 - 重连成功后重新订阅
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Printf("MQTT[%s] 连接成功", name)

		// 重新订阅所有已订阅的主题
		instance.subMu.RLock()
		subs := make(map[string]*subscription)
		for topic, sub := range instance.subscriptions {
			subs[topic] = sub
		}
		instance.subMu.RUnlock()

		if len(subs) > 0 {
			log.Printf("MQTT[%s] 开始重新订阅 %d 个主题...", name, len(subs))
			for topic, sub := range subs {
				token := instance.client.Subscribe(topic, sub.QoS, func(client mqtt.Client, msg mqtt.Message) {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("订阅回调 panic: %v", r)
						}
					}()
					sub.Callback(name, msg.Topic(), msg.Payload())
				})

				if token.WaitTimeout(5*time.Second) && token.Error() == nil {
					log.Printf("MQTT[%s] 重新订阅成功: %s", name, topic)
				} else {
					log.Printf("MQTT[%s] 重新订阅失败: %s, err=%v", name, topic, token.Error())
				}
			}
		}
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	instance.client = client
	m.inst[name] = instance
	log.Printf("MQTT[%s] 初始化完成", name)
	return nil
}

// GetClient 获取原生mqtt client
func (m *Manager) GetClient(name string) (mqtt.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.inst[name]
	if !ok {
		return nil, fmt.Errorf("mqtt实例[%s]不存在", name)
	}
	return inst.client, nil
}

// Exist 判断实例是否存在
func (m *Manager) Exist(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.inst[name]
	return ok
}

// Close 关闭所有连接
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, inst := range m.inst {
		inst.client.Disconnect(200)
		log.Printf("MQTT[%s] 已关闭", name)
	}
}

// ====================== 全局单例 & 初始化 ======================
var GlobalMQTT *Manager

// New 读取配置，批量初始化所有MQTT实例（你原有入口）
func New() error {
	GlobalMQTT = NewManager()
	for name, cfg := range Init.Config.Mqtt {
		err := GlobalMQTT.Add(name, MqttConfig{
			Broker:            cfg.Broker,
			Username:          cfg.Username,
			Password:          cfg.Password,
			ClientID:          cfg.ClientID,
			SetCleanSession:   cfg.SetCleanSession,
			SetAutoReconnect:  cfg.SetAutoReconnect,
			SetConnectTimeout: cfg.SetConnectTimeout,
			SetWriteTimeout:   cfg.SetWriteTimeout,
			SetKeepAlive:      cfg.SetKeepAlive,
		})
		if err != nil {
			log.Fatalf("MQTT连接[%s]初始化失败: %v", name, err)
			return err
		}
	}
	return nil
}

func InitGlobal() {
	GlobalMQTT = NewManager()
}

// ====================== 包级全局收发/订阅函数（你原有风格，保留不动） ======================

// Send 发送消息
func Send(broker string, topic string, data []byte) error {
	if GlobalMQTT == nil {
		return fmt.Errorf("MQTT管理器未初始化")
	}
	GlobalMQTT.mu.RLock()
	inst, ok := GlobalMQTT.inst[broker]
	GlobalMQTT.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	token := inst.client.Publish(topic, 1, false, data)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Subscribe 订阅主题
func Subscribe(broker string, topic string, callback func(broker string, topic string, data []byte)) error {
	if GlobalMQTT == nil {
		return fmt.Errorf("MQTT管理器未初始化")
	}
	GlobalMQTT.mu.RLock()
	inst, ok := GlobalMQTT.inst[broker]
	GlobalMQTT.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	// 保存订阅信息，用于重连后重新订阅
	sub := &subscription{
		Topic:    topic,
		QoS:      1,
		Callback: callback,
	}

	inst.subMu.Lock()
	inst.subscriptions[topic] = sub
	inst.subMu.Unlock()

	token := inst.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		defer func() {
			r := recover()
			if r != nil {
				log.Printf("订阅回调 %v", r)
			}
		}()
		callback(broker, msg.Topic(), msg.Payload())
	})

	// 1. 限时等待订阅完成（3s 超时，避免无限阻塞）
	const subTimeout = 3 * time.Second
	if !token.WaitTimeout(subTimeout) {
		// 超时则移除订阅记录
		inst.subMu.Lock()
		delete(inst.subscriptions, topic)
		inst.subMu.Unlock()
		return fmt.Errorf("订阅超时，broker=%s, topic=%s", broker, topic)
	}

	// 2. 等待结束后，再判断是否有业务错误
	if err := token.Error(); err != nil {
		// 失败则移除订阅记录
		inst.subMu.Lock()
		delete(inst.subscriptions, topic)
		inst.subMu.Unlock()
		log.Printf("订阅失败: broker=%s, topic=%s, err=%v", broker, topic, err)
		return err
	}

	// 3. 全部正常，再打印成功日志
	log.Printf("订阅成功: broker=%s, topic=%s", broker, topic)
	return nil
}

// Subscribe_Close 取消订阅
func Subscribe_Close(broker string, topic string) error {
	if GlobalMQTT == nil {
		return fmt.Errorf("MQTT管理器未初始化")
	}
	GlobalMQTT.mu.RLock()
	inst, ok := GlobalMQTT.inst[broker]
	GlobalMQTT.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	token := inst.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	// 移除订阅记录
	inst.subMu.Lock()
	delete(inst.subscriptions, topic)
	inst.subMu.Unlock()

	log.Printf("取消订阅成功: broker=%s, topic=%s", broker, topic)
	return nil
}
