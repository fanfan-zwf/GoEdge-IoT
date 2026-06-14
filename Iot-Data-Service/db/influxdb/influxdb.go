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

// 写入缓存: key = Tag + 纳秒时间戳，防重复写入
var writeCache sync.Map

// 统一固定测量名
const fixedMeasurement = "point_data"

// 字段映射：按业务类型分配独立字段，彻底解决类型冲突
func getFieldName(typ string) string {
	switch typ {
	case "bool":
		return "bool_val"
	case "int":
		return "int_val"
	case "uint":
		return "uint_val"
	case "string":
		return "string_val"
	default:
		return "string_val"
	}
}

// convertValue 强制类型转换 + 合法性校验
// 规则：int系列统一转int，uint系列统一转uint，bool/string原样保留
func convertValue(typ string, rawVal any) (any, error) {
	switch typ {
	case "bool":
		val, ok := rawVal.(bool)
		if !ok {
			return nil, fmt.Errorf("类型不匹配，期望 bool，实际 %T", rawVal)
		}
		return val, nil

	case "int":
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
			return nil, fmt.Errorf("类型不匹配，期望 int 系列，实际 %T", rawVal)
		}

	case "uint":
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
			return nil, fmt.Errorf("类型不匹配，期望 uint 系列，实际 %T", rawVal)
		}

	case "string":
		val, ok := rawVal.(string)
		if !ok {
			return nil, fmt.Errorf("类型不匹配，期望 string，实际 %T", rawVal)
		}
		return val, nil

	default:
		return nil, fmt.Errorf("不支持的数据类型: %s", typ)
	}
}

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
		// Tag + 时间 去重
		cacheKey := fmt.Sprintf("%s_%d", v.Tag, v.Time.UnixNano())
		if _, exists := writeCache.Load(cacheKey); exists {
			continue
		}

		// 强制类型校验与转换
		realVal, err := convertValue(v.Type, v.Value)
		if err != nil {
			log.Printf("数据转换失败 Tag=%s Type=%s Err=%v", v.Tag, v.Type, err)
			continue
		}

		// 分配对应字段
		fieldKey := getFieldName(v.Type)
		fields := map[string]interface{}{
			fieldKey: realVal,
		}

		// 构建点位：Type、Msg 作为标签
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

	if len(points) > 0 {
		if err := c.writeAPI.WritePoint(ctx, points...); err != nil {
			return fmt.Errorf("写入失败: %w", err)
		}
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

	// 枚举所有业务字段，查询当前tag最新一条数据
	allFields := []string{"bool_val", "int_val", "uint_val", "string_val"}

	for _, tag := range tags {
		var item fullConfig.Value_type
		hasData := false

		for _, field := range allFields {
			flux := fmt.Sprintf(`
				from(bucket: "%s")
				|> range(start: -10y)
				|> filter(fn: (r) => 
					r._measurement == "%s" 
					and r.tag_name == "%s"
					and r._field == "%s"
				)
				|> sort(columns: ["_time"], desc: true)
				|> limit(n: 1)
			`, c.bucket, fixedMeasurement, tag, field)

			res, err := queryAPI.Query(context.Background(), flux)
			if err != nil {
				log.Printf("tag[%s] 字段%s 查询异常: %v", tag, field, err)
				continue
			}

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
				break
			}
		}

		if hasData {
			resultList = append(resultList, item)
		}
	}
	return resultList, nil
}

// ===================== 类型校验工具函数 =====================
func Value_Type_Confirm(Type string, Value any) (v any, err error) {
	return convertValue(Type, Value)
}

// ===================== 全局回调 & 初始化 =====================
func init() {
	db_point.Update_Subscriber(a)
}

// 写入回调
func a(value []fullConfig.Value_type) error {
	index := len(value)
	value = append(value, fullConfig.Value_type{
		Tag:   "//APP//data/influxdb/write_quantity", // 点位名称
		Value: index,                                 // 点位值
		Type:  "int",                                 // 输出类型
		Msg:   "ok",                                  // 状态信息
		Time:  time.Now(),                            // 读取时间
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
