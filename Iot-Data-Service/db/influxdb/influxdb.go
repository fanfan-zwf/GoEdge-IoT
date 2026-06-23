/*
* 日期: 2026.2.22 PM11:15
* 作者: 范范zwf
* 作用: influxdb
 */
package influxdb

import (
	"main/IO/manager/fullConfig"
	"main/Init"
	"main/db/db_point"

	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// 读取范围传递类型
type Read_Scope_type struct {
	Tag        string    // 点位标识
	Value_Type string    // 值类型
	Start_Time time.Time // 开始时间
	End_Time   time.Time // 结束时间
}

// ===================== 连接结构体 =====================
type Connect_struct struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking

	url           string // 地址
	token         string // 令牌
	org           string // 组织
	bucket        string // 存储桶
	write_timeout uint   // 写入超时时间
}

type Connect_interface interface {
	Connect() error
	Close() error
	Packet() error
	initInfluxDB() (err error)
	Write(data []fullConfig.Value_type) (err error)
}

// 全局客户端实例
var c Connect_struct

// 写入缓存: key = Tag + 纳秒时间戳，防重复写入（Tag + Time 作为唯一索引）
var writeCache sync.Map

// 统一固定测量名
const fixedMeasurement = "point_data"

// ===================== 初始化客户端 =====================
func (c *Connect_struct) initInfluxDB() (err error) {
	if c.url == "" {
		return errors.New("URL地址 不能为空")
	}
	if c.token == "" {
		return errors.New("令牌 不能为空")
	}
	if c.org == "" {
		return errors.New("组织 不能为空")
	}
	if c.bucket == "" {
		return errors.New("存储桶 不能为空")
	}

	if c.write_timeout == 0 {
		c.write_timeout = 5000
	}

	c.client = influxdb2.NewClient(c.url, c.token)
	c.writeAPI = c.client.WriteAPIBlocking(c.org, c.bucket)
	return nil
}

func (c *Connect_struct) Connect() error { return nil }
func (c *Connect_struct) Packet() error  { return nil }

// Close 关闭连接
func (c *Connect_struct) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

// getFieldName 根据业务类型分配独立存储字段
// bool/int/uint 统一存入 int_value
// float64 存入 float_value
// string 存入 string_value
func getFieldName(typ string) string {
	switch typ {
	case "bool":
		return "bool_value"
	case "int":
		return "int_value"
	case "uint":
		return "uint_value"
	case "float":
		return "float_value"
	case "string":
		return "string_value"
	default:
		return "string_value"
	}
}

// convertValue 强类型校验 + 强制转换
// 规则：
// 1. bool → int(1/0)
// 2. 所有整型(int8/int16/int32/int64) → int
// 3. 所有无符号整型(uint8/uint16/uint32/uint64) → uint
// 4. 浮点统一 → float64
// 5. 字符串保持原生
func convertValue(expectType string, rawVal any) (any, error) {
	switch expectType {
	case "bool":
		if rawVal == nil {
			return false, nil
		}
		b, ok := rawVal.(bool)
		if !ok {
			return nil, fmt.Errorf("类型不匹配，预期bool，实际类型:%T", rawVal)
		}
		return b, nil
	case "int":
		if rawVal == nil {
			return int(0), nil
		}
		switch v := rawVal.(type) {
		case int:
			return v, nil
		case int8:
			return int(v), nil
		case int16:
			return int(v), nil
		case int32:
			return int(v), nil
		case int64:
			return int(v), nil
		default:
			return nil, fmt.Errorf("类型不匹配，预期int，实际类型:%T", rawVal)
		}

	case "uint":
		if rawVal == nil {
			return uint(0), nil
		}
		switch v := rawVal.(type) {
		case uint:
			return v, nil
		case uint8:
			return uint(v), nil
		case uint16:
			return uint(v), nil
		case uint32:
			return uint(v), nil
		case uint64:
			return uint(v), nil
		default:
			return nil, fmt.Errorf("类型不匹配,预期uint,实际类型:%T", rawVal)
		}

	case "float":
		if rawVal == nil {
			return float64(0.0), nil
		}
		switch v := rawVal.(type) {
		case float64:
			return v, nil
		default:
			return nil, fmt.Errorf("类型不匹配，预期float64，实际类型:%T", rawVal)
		}

	case "string":
		if rawVal == nil {
			return string(""), nil
		}
		s, ok := rawVal.(string)
		if !ok {
			return fmt.Sprintf("%v", rawVal), nil
		}
		return s, nil

	default:
		return fmt.Sprintf("%v", rawVal), nil
	}
}

// ===================== 批量写入 =====================
func (c *Connect_struct) Write(data []fullConfig.Value_type) error {
	if c.client == nil {
		return errors.New("客户端未连接")
	}
	if len(data) == 0 {
		return nil
	}

	var points []*write.Point
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.write_timeout)*time.Millisecond)
	defer cancel()

	for _, v := range data {
		// Tag + 时间戳 作为唯一键，防重复写入
		cacheKey := fmt.Sprintf("%s_%d", v.Tag, v.Time.UnixNano())
		if _, exists := writeCache.Load(cacheKey); exists {
			continue
		}

		// 强制类型转换与校验
		realVal, err := convertValue(v.Type, v.Value)
		if err != nil {
			log.Printf("数据类型转换失败 Tag:%s Type:%s Err:%v", v.Tag, v.Type, err)
			continue
		}

		// 匹配存储字段
		fieldKey := getFieldName(v.Type)
		fields := map[string]interface{}{
			fieldKey: realVal,
		}

		// 构造写入点
		point := influxdb2.NewPoint(
			fixedMeasurement,
			map[string]string{
				"tag_name":  v.Tag,
				"data_type": v.Type,
				"msg":       v.Msg,
			},
			fields,
			v.Time,
		)
		points = append(points, point)
		writeCache.Store(cacheKey, struct{}{})
	}

	// 批量写入
	if len(points) == 0 {
		return nil
	}

	err := c.writeAPI.WritePoint(ctx, points...)
	if err != nil {
		log.Printf("写入失败: %v", err)
		return fmt.Errorf("写入失败: %w", err)
	}

	return nil
}

// ===================== 范围查询 =====================
func (c *Connect_struct) Read(scopes []Read_Scope_type) ([]fullConfig.Value_type, error) {
	if c.client == nil {
		return nil, errors.New("客户端未连接")
	}
	if len(scopes) == 0 {
		return nil, errors.New("读取范围数组为空")
	}

	queryAPI := c.client.QueryAPI(c.org)
	var resultList []fullConfig.Value_type

	for _, scope := range scopes {
		targetTag := scope.Tag
		targetType := scope.Value_Type
		fieldKey := getFieldName(targetType)

		start := scope.Start_Time.Format(time.RFC3339)
		end := scope.End_Time.Format(time.RFC3339)

		flux := fmt.Sprintf(`
			from(bucket: "%s")
			|> range(start: time(v:"%s"), stop: time(v:"%s"))
			|> filter(fn: (r) => 
				r._measurement == "%s" 
				and r.tag_name == "%s" 
				and r.data_type == "%s"
				and r._field == "%s"
			)
			|> sort(columns: ["_time"])
		`, c.bucket, start, end, fixedMeasurement, targetTag, targetType, fieldKey)

		res, err := queryAPI.Query(context.Background(), flux)
		if err != nil {
			return nil, fmt.Errorf("tag[%s] 查询异常: %w", targetTag, err)
		}

		for res.Next() {
			record := res.Record()
			item := fullConfig.Value_type{
				Tag:   record.ValueByKey("tag_name").(string),
				Type:  record.ValueByKey("data_type").(string),
				Msg:   record.ValueByKey("msg").(string),
				Value: record.Value(),
				Time:  record.Time(),
			}
			resultList = append(resultList, item)
		}

		if err := res.Err(); err != nil {
			res.Close()
			return nil, fmt.Errorf("tag[%s] 解析结果异常: %w", targetTag, err)
		}
		res.Close()
	}
	return resultList, nil
}

// ===================== 查询每个Tag 最后一条数据 =====================
func (c *Connect_struct) QueryLast(tags []string) ([]fullConfig.Value_type, error) {
	if c.client == nil {
		return nil, errors.New("客户端未连接")
	}
	if len(tags) == 0 {
		return nil, errors.New("tag列表不能为空")
	}

	queryAPI := c.client.QueryAPI(c.org)
	var resultList []fullConfig.Value_type

	for _, tag := range tags {
		flux := fmt.Sprintf(`
			from(bucket: "%s")
			|> range(start: -10y)
			|> filter(fn: (r) => 
				r._measurement == "%s" 
				and r.tag_name == "%s"
				and (r._field == "bool_value" or r._field == "int_value" or r._field == "uint_value" or r._field == "float_value" or r._field == "string_value")
			)
			|> sort(columns: ["_time"], desc: true)
			|> limit(n: 1)
		`, c.bucket, fixedMeasurement, tag)

		res, err := queryAPI.Query(context.Background(), flux)
		if err != nil {
			return nil, fmt.Errorf("tag[%s] 查询最后一条失败: %w", tag, err)
		}

		var item fullConfig.Value_type
		hasData := false
		for res.Next() {
			record := res.Record()
			item = fullConfig.Value_type{
				Tag:   record.ValueByKey("tag_name").(string),
				Type:  record.ValueByKey("data_type").(string),
				Msg:   record.ValueByKey("msg").(string),
				Value: record.Value(),
				Time:  record.Time(),
			}
			hasData = true
		}
		res.Close()

		if hasData {
			resultList = append(resultList, item)
		}
	}
	return resultList, nil
}

// ===================== 全局回调 & 初始化 =====================
func init() {
	db_point.Update_Subscriber(a)
}

// 写入回调
func a(value []fullConfig.Value_type) error {
	index := len(value)
	value = append(value, fullConfig.Value_type{
		Tag:   Init.Config.Influxdb.Write_Quantity_Tag,
		Value: index,
		Type:  "int",
		Msg:   "ok",
		Time:  time.Now(),
	})

	err := c.Write(value)
	if err != nil {
		log.Printf("influxdb 写入回调异常: %v", err)
	}
	return nil
}

// New 初始化全局InfluxDB实例
func New() error {
	c = Connect_struct{
		url:           Init.Config.Influxdb.Url,
		token:         Init.Config.Influxdb.Token,
		org:           Init.Config.Influxdb.Org,
		bucket:        Init.Config.Influxdb.Bucket,
		write_timeout: 5000,
	}
	return c.initInfluxDB()
}
