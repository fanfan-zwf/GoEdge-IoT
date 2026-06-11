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

// 统一固定测量名（所有数据存在同一个measurement，不使用业务Tag）
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

// ===================== 批量写入 =====================
// 入参: []Value_type
// 规则: Tag + Time 作为唯一键，重复数据自动跳过
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
		// 唯一键：Tag + 纳秒时间戳
		cacheKey := fmt.Sprintf("%s_%d", v.Tag, v.Time.UnixNano())
		if _, exists := writeCache.Load(cacheKey); exists {
			continue
		}

		// 构造数据点：全部独立字段/标签，不做任何拼接
		point := influxdb2.NewPoint(
			fixedMeasurement, // 固定测量名
			map[string]string{
				"tag_name":  v.Tag,  // 点位名称
				"data_type": v.Type, // 值类型
				"msg":       v.Msg,  // 状态信息
			},
			map[string]interface{}{
				"field_value": v.Value, // 点位值
			},
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
// 入参: []Read_Scope_type
// 按 Tag + 时间范围 查询，返回 []Value_type
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
		start := scope.Start_Time.Format(time.RFC3339)
		end := scope.End_Time.Format(time.RFC3339)

		// Flux: 根据 tag_name(Tag) + data_type(Type) + 时间范围 查询
		flux := fmt.Sprintf(`
			from(bucket: "%s")
			|> range(start: time(v:"%s"), stop: time(v:"%s"))
			|> filter(fn: (r) => 
				r._measurement == "%s" 
				and r.tag_name == "%s" 
				and r.data_type == "%s"
			)
			|> sort(columns: ["_time"])
		`, c.bucket, start, end, fixedMeasurement, targetTag, targetType)

		res, err := queryAPI.Query(context.Background(), flux)
		if err != nil {
			return nil, fmt.Errorf("tag[%s] 查询异常: %w", targetTag, err)
		}

		// 解析结果组装为 Value_type
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
// 入参: []string 点位Tag列表
// 返回: []Value_type（按传入tag顺序返回，无数据则不包含该条）
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
			|> filter(fn: (r) => r._measurement == "%s" and r.tag_name == "%s")
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

		// 有数据才加入结果切片
		if hasData {
			resultList = append(resultList, item)
		}
	}

	return resultList, nil
}

// ===================== 类型校验工具函数 =====================
func Value_Type_Confirm(Type string, Value any) (v any, err error) {
	switch Type {
	case "bool":
		if val, ok := Value.(bool); ok {
			return val, nil
		}
	case "int8":
		if val, ok := Value.(int8); ok {
			return val, nil
		}
	case "uint8":
		if val, ok := Value.(uint8); ok {
			return val, nil
		}
	case "int16":
		if val, ok := Value.(int16); ok {
			return val, nil
		}
	case "uint16":
		if val, ok := Value.(uint16); ok {
			return val, nil
		}
	case "int32":
		if val, ok := Value.(int32); ok {
			return val, nil
		}
	case "uint32":
		if val, ok := Value.(uint32); ok {
			return val, nil
		}
	case "int64":
		if val, ok := Value.(int64); ok {
			return val, nil
		}
	case "uint64":
		if val, ok := Value.(uint64); ok {
			return val, nil
		}
	case "int":
		if val, ok := Value.(int); ok {
			return val, nil
		}
	case "uint":
		if val, ok := Value.(uint); ok {
			return val, nil
		}
	case "float32":
		if val, ok := Value.(float32); ok {
			return val, nil
		}
	case "float", "float64":
		if val, ok := Value.(float64); ok {
			return val, nil
		}
	}
	return nil, fmt.Errorf("值类型不匹配，期望: %s", Type)
}

// ===================== 全局回调 & 初始化 =====================
func init() {
	db_point.Update_Subscriber(a)
}

// 写入回调
func a(value []fullConfig.Value_type) error {
	if err := c.Write(value); err != nil {
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
