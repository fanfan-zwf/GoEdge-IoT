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

type mqttInstance struct {
	client mqtt.Client
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

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("MQTT[%s] 连接断开: %v", name, err)
	})
	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		log.Printf("MQTT[%s] 尝试重连...", name)
	})

	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Printf("MQTT[%s] 连接成功", name)
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.inst[name] = &mqttInstance{client: client}
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

	token := inst.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("订阅回调 panic: %v", r)
			}
		}()
		callback(broker, msg.Topic(), msg.Payload())
	})
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

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

	log.Printf("取消订阅成功: broker=%s, topic=%s", broker, topic)
	return nil
}
