/*
* 日期: 2026.2.22 PM11:15
* 作者: 范范zwf
* 作用: influxdb
 */

package influxdb

import (
	"main/Init"
	"main/db/db_point"

	"context"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// CEL6GU0n0-lsU2SZG6TB3pLLsE6zeWbDnzce3NuIy6x1tfQdPoP63MmRJbMDL1TJxYY_LNEY8MLN5OCh1GHODw==

// 读取具体传递时间
type Read_Specific_type struct {
	Tag        string    // 点位标识
	Value_Type string    // 值类型
	Time       time.Time //  时间
}

// 读取范围传递类型
type Read_Scope_type struct {
	Tag        string    // 点位标识
	Value_Type string    // 值类型
	Start_Time time.Time // 开始时间
	End_Time   time.Time // 结束时间
}

type Read_Scope_Value_Data_type struct {
	Value any
	Msg   string
	Time  time.Time
}

// 读取数据返回类型
type Read_Scope_Data_type struct {
	Tag        string // 点位标识
	Value_Type string // 值类型
	Data       []Read_Scope_Value_Data_type
}

/*******************驱动接口配置*******************/

// 定义一个结构体
type Connect_struct struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking

	url    string // 地址
	token  string // 令牌
	org    string // 组织
	bucket string // 存储桶

	// 写入
	write_timeout uint // 写入超时时间
}

// 定义接口
type Connect_interface interface {
	Connect() error                                  // 连接
	Close() error                                    // 关闭连接
	Packet() error                                   // 组包
	initInfluxDB() (err error)                       // 初始化InfluxDB客户端（程序启动时执行1次）
	Write(data []db_point.Db_Value_type) (err error) // 批量写入函数

}

// 初始化InfluxDB客户端（程序启动时执行1次）
func (c *Connect_struct) initInfluxDB() (err error) {
	if c.url == "" {
		err = fmt.Errorf("URL地址 不能为空")
		return
	}
	if c.token == "" {
		err = fmt.Errorf("令牌 不能为空")
		return
	}
	if c.org == "" {
		err = fmt.Errorf("组织 不能为空")
		return
	}
	if c.bucket == "" {
		err = fmt.Errorf("存储桶 不能为空")
		return
	}

	// 设置默认写入超时时间
	if c.write_timeout == 0 {
		c.write_timeout = 5000
	}

	// 创建客户端（全局复用，不要每次创建）
	c.client = influxdb2.NewClient(c.url, c.token)
	// 初始化阻塞式写入客户端（关联org和bucket）
	c.writeAPI = c.client.WriteAPIBlocking(c.org, c.bucket)
	// 若用异步写入：writeAPIAsync = influxClient.WriteAPI(org, bucket)

	return
}

// 批量写入函数
func (c *Connect_struct) Write(data []db_point.Db_Value_type) (err error) {
	if c.client == nil || len(data) == 0 {
		err = fmt.Errorf("客户端未连接")
		return
	}

	var points []*write.Point

	for _, v := range data {

		point := influxdb2.NewPoint(
			v.Tag, // 测量名，可根据业务修改
			map[string]string{
				"_msg": v.Msg,
			}, // 标签：点位ID唯一标识
			map[string]interface{}{v.Type + "_value": v.Value}, // 字段：存储int/float/string值
			v.Time, // 时间戳
		)
		points = append(points, point)
	}

	// 批量写入数据，设置5秒超时
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.write_timeout)*time.Millisecond,
	)
	defer cancel()
	err = c.writeAPI.WritePoint(ctx, points...)

	return
}

// 批量读取函数（严格匹配你的结构体定义）
// 入参：Read_Scope_type数组 → 出参：Read_Scope_Data_type数组 + 全局错误
func (c *Connect_struct) Read(scopes []Read_Scope_type) (readResults []Read_Scope_Data_type, err error) {
	// 前置校验
	if c.client == nil {
		err = fmt.Errorf("influxdb客户端未连接")
		return
	}
	if len(scopes) == 0 {
		err = fmt.Errorf("读取范围数组为空")
		return
	}

	// 初始化返回结果集
	// 获取InfluxDB查询API（v2必需）
	queryAPI := c.client.QueryAPI(c.org)

	// 遍历每个读取范围，逐个查询
	for _, scope := range scopes {
		// 初始化当前点位的返回结构
		scopeResult := Read_Scope_Data_type{
			Tag:        scope.Tag,
			Value_Type: scope.Value_Type,
			Data:       []Read_Scope_Value_Data_type{}, // 初始化为空切片
		}

		// 1. 基础参数校验
		if scope.Tag == "" {
			err = fmt.Errorf("点位标识Tag不能为空(某条读取范围)")
			return
		}
		if scope.Value_Type == "" {
			err = fmt.Errorf("值类型Value_Type不能为空(Tag:%s)", scope.Tag)
			return
		}
		if scope.Start_Time.After(scope.End_Time) {
			err = fmt.Errorf("开始时间晚于结束时间(Tag:%s)", scope.Tag)
			return
		}
		// 校验支持的类型
		switch scope.Value_Type {
		case "int", "float", "string", "bool":
		default:
			err = fmt.Errorf("不支持的值类型：%s(Tag:%s),仅支持int/float/string/bool", scope.Value_Type, scope.Tag)
			return
		}

		// 2. 拼接字段名（和你的写入逻辑完全匹配：Value_Type + "_value"）
		fieldName := scope.Value_Type + "_value"

		// 3. 构造Flux查询语句（InfluxDB v2标准查询语法）
		fluxQuery := fmt.Sprintf(`
			from(bucket: "%s")
				|> range(start: time(v: "%s"), stop: time(v: "%s"))
				|> filter(fn: (r) => r._measurement == "%s")
				|> filter(fn: (r) => r._field == "%s")
				|> sort(columns: ["_time"], desc: false) // 按时间升序排列
				|> keep(columns: ["_time", "_value", "_msg"])    // 只保留需要的字段，提升性能
		`,
			c.bucket,                              // 你的InfluxDB桶名（Connect_struct需包含该字段）
			scope.Start_Time.Format(time.RFC3339), // 标准化时间格式
			scope.End_Time.Format(time.RFC3339),
			scope.Tag, // 测量名 = 点位Tag（和写入一致）
			fieldName, // 字段名 = 类型+_value（如int_value）
		)

		// 4. 执行查询
		var result *api.QueryTableResult
		result, err = queryAPI.Query(context.Background(), fluxQuery)
		if err != nil {
			err = fmt.Errorf("查询失败(Tag:%s:%w", scope.Tag, err)
			return
		}
		defer result.Close() // 确保关闭结果集，释放资源

		// 5. 解析查询结果到结构体
		for result.Next() {
			record := result.Record()
			if record == nil {
				continue
			}

			// 根据Value_Type做类型断言，保证值类型准确
			var val any
			val, err = Value_Type_Confirm(scope.Value_Type, record.Value())
			if err != nil {
				log.Print(err)
				continue
			}

			// 4.1 提取_msg标签值（对应Msg字段）
			msgVal, ok := record.ValueByKey("_msg").(string)
			if !ok {
				// 兼容_msg为空的情况，赋值为空字符串
				msgVal = ""
			}

			// 追加单条数据到结果
			scopeResult.Data = append(scopeResult.Data, Read_Scope_Value_Data_type{
				Time:  record.Time(),
				Value: val,
				Msg:   msgVal,
			})
		}

		// 6. 检查结果解析过程中的错误
		err = result.Err()
		if err != nil {
			err = fmt.Errorf("解析查询结果失败(Tag:%s):%w", scope.Tag, err)
			return
		}

		// 7. 将当前点位的结果加入总结果集
		readResults = append(readResults, scopeResult)
	}

	return
}

var c Connect_struct

func init() {
	db_point.Update_Subscriber(a)
}

func a(value []db_point.Update_Value_type) error {
	var db_value []db_point.Db_Value_type
	for _, v := range value {
		db_value = append(db_value, v.Db_Value_type)
	}

	err := c.Write(db_value)
	if err != nil {
		log.Print(err.Error())
	}

	return nil
}

func New() (err error) {
	c = Connect_struct{
		url:    Init.Config.Influxdb.Url,
		token:  Init.Config.Influxdb.Token,
		org:    Init.Config.Influxdb.Org,
		bucket: Init.Config.Influxdb.Bucket,
	}
	err = c.initInfluxDB()

	return
}

// 这个是把读取返回类型是any做以下确认是否和指定类型一致
// 传入：Type：期望类型，Value：值
// 返回：v：值，err：错误
func Value_Type_Confirm(Type string, Value any) (v any, err error) {
	switch Type {
	case "bool":
		var bool_value bool
		bool_value, ok := Value.(bool)
		if !ok {
			err = fmt.Errorf("类型不是 bool")
		} else {
			v = bool_value
		}
	case "int8":
		var int8_value int8
		int8_value, ok := Value.(int8)
		if !ok {
			err = fmt.Errorf("类型不是 int8")
		} else {
			v = int8_value
		}
	case "uint8":
		var uint8_value uint8
		uint8_value, ok := Value.(uint8)
		if !ok {
			err = fmt.Errorf("类型不是 uint8")
		} else {
			v = uint8_value
		}
	case "int16":
		var int16_value int16
		int16_value, ok := Value.(int16)
		if !ok {
			err = fmt.Errorf("类型不是 int16")
		} else {
			v = int16_value
		}
	case "uint16":
		var uint16_value uint16
		uint16_value, ok := Value.(uint16)
		if !ok {
			err = fmt.Errorf("类型不是 uint16")
		} else {
			v = uint16_value
			return
		}
	case "int32":
		var int32_value int32
		int32_value, ok := Value.(int32)
		if !ok {
			err = fmt.Errorf("类型不是 int16")
		} else {
			v = int32_value
		}
	case "uint32":
		var uint32_value uint32
		uint32_value, ok := Value.(uint32)
		if !ok {
			err = fmt.Errorf("类型不是 uint16")
		} else {
			v = uint32_value
		}
	case "int64":
		var int64_value int64
		int64_value, ok := Value.(int64)
		if !ok {
			err = fmt.Errorf("类型不是 int64")
		} else {
			v = int64_value
		}
	case "uint64":
		var uint64_value uint64
		uint64_value, ok := Value.(uint64)
		if !ok {
			err = fmt.Errorf("类型不是 uint64")
		} else {
			v = uint64_value
		}
	case "int":
		var int_value int
		int_value, ok := Value.(int)
		if !ok {
			err = fmt.Errorf("类型不是 int")
		} else {
			v = int_value
		}
	case "uint":
		var uint_value uint
		uint_value, ok := Value.(uint)
		if !ok {
			err = fmt.Errorf("类型不是 uint")
		} else {
			v = uint_value
		}
	case "float32":
		var float32_value float32
		float32_value, ok := Value.(float32)
		if !ok {
			err = fmt.Errorf("类型不是 float32")
		} else {
			v = float32_value
		}
	case "float64", "float":
		var float64_value float64
		float64_value, ok := Value.(float64)
		if !ok {
			err = fmt.Errorf("类型不是 float64")
		} else {
			v = float64_value
		}
	default:
		err = fmt.Errorf("未知类型")
	}

	return
}
