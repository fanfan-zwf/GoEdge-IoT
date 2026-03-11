package Rec_Yun_Cloud

import (
	"encoding/json"
	"main/cloud"
	"main/db/db_point"
	"main/db/mysql"
	my_mysql "main/db/mysql"

	"fmt"
	"log"
	"time"

	mqtt_go "github.com/eclipse/paho.mqtt.golang"
)

// 发送和接受的数据结构体
type data_type struct {
	version string
	data    []db_point.Db_Value_type
}

/*******************驱动配置*******************/

type Currently_Points_type struct {
	Topic string // 主题
	Qos   uint8
}

// 连接
type Config_type struct {
	URI []string // IP地址

	AutoReconnect        bool // 断连后自动重连
	ConnectRetry         bool // 首次连接失败后重试
	ConnectTimeout       uint // 单次连接url的超时时间
	ConnectRetryInterval uint // 重连 / 重试的时间间隔

	ClientID   string // 客户端id
	User       string // 名称
	Passwd     string // 密码
	AES_Passwd string // AES加密密码 可选，默认不加密
}

/*******************驱动连接*******************/

// 定义一个结构体
type Connect_struct struct {
	Drive          my_mysql.Drive_Config_type // 通信参数结构体
	Connect_Config Config_type                // 连接配置

	Points         []mysql.Points_Config_type
	Points_Tag_Map map[string]int

	Client mqtt_go.Client // 连接
	Esc    chan bool      // 循环读取退出

	Packets []Currently_Points_type
}

// 定义接口
type Connect_interface interface {
	ConnectionLostHandler(client mqtt_go.Client, err error)    // MQTT断开重新连接
	MessageHandler(client mqtt_go.Client, msg mqtt_go.Message) // 订阅定义消息处理器
	SubscribeSingleTopic()                                     // 订阅单个主题
	MQTTCommand_Publish() error                                // 发送消息
	Connect() error                                            // 连接
	Close() error                                              // 关闭连接
	Packet() error                                             // 组包

	// 输入： Tag // 点位标识
	// 输出： Collect_Id：设备id err错误
	Search_Points__CollectId(Tag string) (Point my_mysql.Points_Config_type, err error)
}

// 初始化
func (c *Connect_struct) MQTTCommand_Publish_Init() (err error) {

	return
}

// 输入： Tag 点位标识
// 输出： Point 点位配置 err 错误
func (c *Connect_struct) Search_Points__CollectId(Tag string) (Point my_mysql.Points_Config_type, err error) {
	index, ok := c.Points_Tag_Map[Tag]
	if !ok {
		err = fmt.Errorf("不存在的点位标识")
		return
	}
	if index < 0 || index >= len(c.Points) {
		err = fmt.Errorf("点位下标越界, index: %d, 切片长度: %d", index, len(c.Points))
		return
	}
	Point = c.Points[index]
	return
}

// 组包
func (c *Connect_struct) Packet() (err error) {
	c.Points_Tag_Map = make(map[string]int)
	var Packets []Currently_Points_type
	seen := make(map[Currently_Points_type]struct{})
	for i, v := range c.Points {

		var currently Currently_Points_type
		currently, err = Currently_Point(v.Config)
		if err != nil {
			return
		}
		c.Points_Tag_Map[v.Tag] = i

		_, exists := seen[currently]
		if !exists {
			seen[currently] = struct{}{}
			Packets = append(Packets, currently)
		}
	}
	c.Packets = Packets
	return
}

// MQTT断开重新连接
func (c *Connect_struct) ConnectionLostHandler(client mqtt_go.Client, err error) {
	if err != nil {
		log.Printf("INFO 驱动ID:%d,驱动名称：%s 重新连接:%v", c.Drive.Id, c.Drive.Name, err)
	}

	select {
	case <-time.After(5 * time.Second): // 情况1：5秒超时
		fmt.Println("5秒超时到达")
	case _, ok := <-c.Esc: // 情况2：监听 done 通道是否关闭或有数据
		if !ok { // 如果 ok 为 false，说明通道被关闭
			return
		}
		// 如果 ok 为 true，说明收到了数据，可以根据需要处理
		// 此示例中，我们只关心通道是否关闭，所以可以忽略数据
	}

	token := client.Connect()
	err = token.Error()

	if token.Wait() && err != nil {
		log.Printf("ERROR 驱动ID:%d,驱动名称：%s 重新连接失败:%v", c.Drive.Id, c.Drive.Name, err)
	} else {
		log.Printf("INFO 驱动ID:%d,驱动名称：%s 重新连接成功", c.Drive.Id, c.Drive.Name)
	}
}

// 订阅定义消息处理器
func (c *Connect_struct) MessageHandler(client mqtt_go.Client, msg mqtt_go.Message) {
	Topic := msg.Topic()

	var err error
	defer func(err error) {
		if err != nil {
			log.Printf("ERROR %s", err.Error())
		}
	}(err)

	Payload := msg.Payload()

	var receive_type []byte
	receive_type, err = cloud.Receive__CRC32_Aes_Gzip(Payload, c.Connect_Config.AES_Passwd)
	if err != nil {
		log.Print(err)
		return
	}

	var (
		Change_Value_List []db_point.Db_Value_type
		db_value_List     []db_point.Db_Value_type
	)

	err = json.Unmarshal([]byte(receive_type), &Change_Value_List)
	if err != nil {
		err = fmt.Errorf("反序列化失败: %v", err)
		return
	}

	for _, Change_Value := range Change_Value_List {

		Change_Value.Value, err = db_point.ConvertValueByType(Change_Value.Value, Change_Value.Type)
		if err != nil {
			Change_Value.Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
		}

		var consistency bool
		consistency, err = db_point.Drive_Type_Map__Consistency(
			Change_Value.Tag,
			package_drive_type,
			"",
			Change_Value.Type,
		)

		if err != nil {
			log.Print(err)
			err = nil
			continue
		}

		if !consistency {
			log.Printf("ERROR 接收到信息和配置不一致")
			continue
		}

		var Point my_mysql.Points_Config_type
		Point, err = c.Search_Points__CollectId(Change_Value.Tag)
		if err != nil {
			log.Print(err)
			continue
		}

		var topic_cfg string
		topic_cfg, err = Topic_Config(Point.Config)
		if err != nil {
			log.Print(err)
			continue
		}

		if topic_cfg != Topic {
			err = fmt.Errorf("订阅的主题与配置的主题不一致 订阅主题:%s 配置点位主题:%s 配置点位标识:%s", Topic, topic_cfg, Point.Tag)
			log.Print(err)
			continue
		}

		if Change_Value.Tag != Point.Tag {
			err = fmt.Errorf("接收的点位标识与配置的点位标识不一致 接收点位标识:%s 配置点位标识:%s", Change_Value.Tag, Point.Tag)
			log.Print(err)
			continue
		}

		if !(Point.RW_Cancel == "W/R" || Point.RW_Cancel == "R") {
			err = fmt.Errorf("点位不是一个可以读取的点位 读取模式:%s 配置点位标识:%s", Point.RW_Cancel, Point.Tag)
			log.Print(err)
			continue
		}

		db_value_List = append(db_value_List, Change_Value)
	}

	err = db_point.Db_Publisher(db_value_List)

}

// 订阅单个主题
func (c *Connect_struct) SubscribeSingleTopic() (err error) {

	// err = c.Packet()

	for _, v := range c.Packets {
		token := c.Client.Subscribe(v.Topic, v.Qos, nil)
		if token.Wait() && token.Error() != nil {
			err = fmt.Errorf("ERROR 驱动ID:%d,驱动名称：%s 订阅主题 %s 失败: %v", c.Drive.Id, c.Drive.Name, v.Topic, token.Error())
			log.Print(err.Error())
		}
	}
	return
}

// 变化 发送消息
func (c *Connect_struct) MQTTCommand_Publish() (err error) {

	// // 将结构体转换为 JSON
	// jsonData, err := json.Marshal(cache)
	// if err != nil {
	// 	log.Printf("ERROR %v", err)
	// 	return
	// }

	// token := c.Client.Publish(c.Drive.Config.Topic+"/Timing", c.Drive.Config.Qos, false, string(jsonData))
	// token.Wait()
	return
}

// 连接
func (c *Connect_struct) Connect() (err error) {
	if c.Drive.Type != package_drive_type {
		log.Printf("ERROR Yun_Cloud配置类型不正确 驱动ID:%d,驱动名称：%s", c.Drive.Id, c.Drive.Name)
		return fmt.Errorf("配置类型不正确")
	}

	var Connect_Config Config_type
	err = json.Unmarshal([]byte(c.Drive.Config), &Connect_Config)
	if err != nil {
		return
	}
	c.Connect_Config = Connect_Config

	c.Esc = make(chan bool)

	opts := mqtt_go.NewClientOptions()

	for _, v := range Connect_Config.URI {
		opts.AddBroker(v)
	}
	opts.SetClientID(Connect_Config.ClientID)
	opts.SetUsername(Connect_Config.User)
	opts.SetPassword(Connect_Config.Passwd)
	opts.SetDefaultPublishHandler(c.MessageHandler)
	opts.SetConnectionLostHandler(c.ConnectionLostHandler)
	// opts.OnConnect = connectHandler
	// opts.OnConnectionLost = connectLostHandler

	opts.SetAutoReconnect(Connect_Config.AutoReconnect)                                                 // 断连后自动重连
	opts.SetConnectRetry(Connect_Config.ConnectRetry)                                                   // 首次连接失败后重试
	opts.SetConnectTimeout(time.Duration(Connect_Config.ConnectTimeout) * time.Millisecond)             // 单次连接url的超时时间
	opts.SetConnectRetryInterval(time.Duration(Connect_Config.ConnectRetryInterval) * time.Millisecond) // 重连 / 重试的时间间隔

	c.Client = mqtt_go.NewClient(opts)

	token := c.Client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Printf("WARNING 驱动ID:%d,驱动名称：%s,连接错误:%v", c.Drive.Id, c.Drive.Name, token.Error())
	}

	c.SubscribeSingleTopic() // 订阅单个主题
	c.MQTTCommand_Publish()  // 发送消息

	return nil
}

// 关闭连接
func (c *Connect_struct) Close() error {
	close(c.Esc) // 发送关闭消息

	if c.Client != nil {
		c.Client.Disconnect(250)
	}
	if !c.Client.IsConnected() {
		log.Print("INFO ", c.Drive.Id, "关闭成功")
		c.Client = nil
		return nil
	}

	log.Print("ERROR ", c.Drive.Id, "关闭失败")
	return fmt.Errorf("关闭失败")
}

// 标准处理
func Qos_Config(config string) (qos uint8, err error) {
	var Currently Currently_Points_type
	err = json.Unmarshal([]byte(config), &Currently)
	qos = Currently.Qos
	return
}

func Topic_Config(config string) (Topic string, err error) {
	var Currently Currently_Points_type
	err = json.Unmarshal([]byte(config), &Currently)
	Topic = Currently.Topic
	return
}

func Currently_Point(config string) (Currently Currently_Points_type, err error) {
	err = json.Unmarshal([]byte(config), &Currently)
	return
}
