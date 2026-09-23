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
	"sync"

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

	// 批量写入缓冲区
	bufferMu      sync.Mutex
	buffer        []fullConfig.Value_type
	bufferSize    int           // 缓冲区大小阈值
	flushInterval time.Duration // 刷新间隔
	lastFlushTime time.Time     // 上次刷新时间
	stopChan      chan struct{} // 停止信号
	isRunning     bool          // 后台刷新协程是否运行
}

// 定义接口
type Connect_interface interface {
	Connect() error                                 // 连接
	Close() error                                   // 关闭连接
	Packet() error                                  // 组包
	initInfluxDB() (err error)                      // 初始化InfluxDB客户端（程序启动时执行1次）
	Write(data []fullConfig.Value_type) (err error) // 批量写入函数

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

	// 设置批量写入缓冲区参数
	if c.bufferSize == 0 {
		c.bufferSize = 100 // 默认缓冲区大小：100条数据
	}
	if c.flushInterval == 0 {
		c.flushInterval = 2 * time.Second // 默认刷新间隔：2秒
	}

	// 创建客户端（全局复用，不要每次创建）
	c.client = influxdb2.NewClient(c.url, c.token)
	// 初始化阻塞式写入客户端（关联org和bucket）
	c.writeAPI = c.client.WriteAPIBlocking(c.org, c.bucket)
	// 若用异步写入：writeAPIAsync = influxClient.WriteAPI(org, bucket)

	return
}

// Close 关闭连接
func (c *Connect_struct) Close() error {
	// 停止后台刷新协程
	c.stopBackgroundFlush()

	// 刷新剩余数据
	c.flushBuffer()

	if c.client != nil {
		c.client.Close()
	}

	// 清理所有缓存
	writeCache.Range(func(key, value interface{}) bool {
		writeCache.Delete(key)
		return true
	})

	return nil
}

// 批量写入函数（使用缓冲区机制）
func (c *Connect_struct) Write(data []fullConfig.Value_type) (err error) {
	if c.client == nil || len(data) == 0 {
		err = fmt.Errorf("客户端未连接")
		return
	}

	// 使用缓冲区机制，将数据添加到缓冲区
	c.addToBuffer(data)
	return nil
}

// startBackgroundFlush 启动后台定时刷新协程
func (c *Connect_struct) startBackgroundFlush() {
	c.bufferMu.Lock()
	if c.isRunning {
		c.bufferMu.Unlock()
		return
	}
	c.isRunning = true
	c.stopChan = make(chan struct{})
	c.lastFlushTime = time.Now()
	c.bufferMu.Unlock()

	go func() {
		ticker := time.NewTicker(c.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.flushBuffer()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// stopBackgroundFlush 停止后台刷新协程
func (c *Connect_struct) stopBackgroundFlush() {
	c.bufferMu.Lock()
	if !c.isRunning {
		c.bufferMu.Unlock()
		return
	}
	close(c.stopChan)
	c.isRunning = false
	c.bufferMu.Unlock()
}

// flushBuffer 刷新缓冲区，将数据批量写入 InfluxDB
func (c *Connect_struct) flushBuffer() error {
	c.bufferMu.Lock()
	if len(c.buffer) == 0 {
		c.bufferMu.Unlock()
		return nil
	}

	// 取出缓冲区数据
	data := make([]fullConfig.Value_type, len(c.buffer))
	copy(data, c.buffer)
	c.buffer = c.buffer[:0] // 清空缓冲区
	c.lastFlushTime = time.Now()
	c.bufferMu.Unlock()

	// 执行实际写入
	if err := c.doWrite(Init.Config.APP.Label, data); err != nil {
		log.Printf("InfluxDB 批量写入失败，数据量: %d, 错误: %v", len(data), err)
		// 写入失败时，将数据重新放回缓冲区（可选策略）
		c.bufferMu.Lock()
		c.buffer = append(c.buffer, data...)
		c.bufferMu.Unlock()
		return err
	}

	// 定期清理过期的缓存键，防止内存泄漏
	c.cleanupWriteCache()

	return nil
}

// cleanupWriteCache 清理过期的写入缓存键
func (c *Connect_struct) cleanupWriteCache() {
	cacheCleanupMu.Lock()
	defer cacheCleanupMu.Unlock()

	now := time.Now()
	if now.Sub(lastCleanupTime) < cacheMaxAge/2 {
		// 距离上次清理时间不足一半，跳过
		return
	}

	lastCleanupTime = now
	cutoffTime := now.Add(-cacheMaxAge).UnixNano()

	// 遍历并删除过期的缓存键
	writeCache.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			// 从 cacheKey 格式 "pointId_timestamp" 中提取时间戳部分
			for i := len(keyStr) - 1; i >= 0; i-- {
				if keyStr[i] == '_' {
					if i < len(keyStr)-1 {
						// 提取时间戳字符串
						tsStr := keyStr[i+1:]
						var ts int64
						fmt.Sscanf(tsStr, "%d", &ts)
						if ts < cutoffTime {
							writeCache.Delete(key)
						}
					}
					break
				}
			}
		}
		return true
	})
}

// addToBuffer 添加数据到缓冲区，达到阈值时自动刷新
func (c *Connect_struct) addToBuffer(data []fullConfig.Value_type) {
	c.bufferMu.Lock()
	c.buffer = append(c.buffer, data...)

	// 检查是否达到缓冲区大小阈值
	shouldFlush := len(c.buffer) >= c.bufferSize
	c.bufferMu.Unlock()

	// 如果达到阈值，立即刷新
	if shouldFlush {
		c.flushBuffer()
	}
}

// doWrite 实际执行批量写入操作（内部方法）
func (c *Connect_struct) doWrite(deviceID string, data []fullConfig.Value_type) error {
	if c.client == nil {
		return fmt.Errorf("客户端未连接")
	}
	if len(data) == 0 {
		return nil
	}

	var points []*write.Point
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.write_timeout)*time.Millisecond)
	defer cancel()

	for _, v := range data {
		// Tag + 时间戳 作为唯一键，防重复写入
		cacheKey := fmt.Sprintf("%d_%d", v.PointId, v.Time.UnixNano())
		if _, exists := writeCache.Load(cacheKey); exists {
			continue
		}

		// measurement: 设备唯一标识符（APP.SN）
		// tags: 点位 ID（PointId）
		// fields: value_bool/value_int/value_uint/value_float/value_string（按类型分字段）, msg（状态）
		fields := make(map[string]interface{})
		fields["msg"] = v.Msg // 状态信息

		// 根据实际类型存储到不同字段，避免 InfluxDB 类型冲突
		switch val := v.Value.(type) {
		case bool:
			fields["value_bool"] = val
		case int, int8, int16, int32, int64:
			fields["value_int"] = val
		case uint, uint8, uint16, uint32, uint64:
			fields["value_uint"] = val
		case float32, float64:
			fields["value_float"] = val
		case string:
			fields["value_string"] = val
		default:
			// 其他类型转为字符串
			fields["value_string"] = fmt.Sprintf("%v", v.Value)
		}

		point := influxdb2.NewPoint(
			fmt.Sprintf("device_%s", deviceID), // measurement: 设备标识
			map[string]string{
				"point_id": fmt.Sprintf("%d", v.PointId), // tag: 点位ID
			},
			fields,
			v.Time, // 时间戳
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

// QueryByPointIdAndTime 按点位ID和时间范围查询历史数据
func QueryByPointIdAndTime(pointId uint, startTime, endTime time.Time, page, pageSize uint) ([]map[string]interface{}, int64, error) {
	return c.queryByPointIdAndTime(pointId, startTime, endTime, page, pageSize)
}

// queryByPointIdAndTime 内部实现
func (c *Connect_struct) queryByPointIdAndTime(pointId uint, startTime, endTime time.Time, page, pageSize uint) ([]map[string]interface{}, int64, error) {
	if c.client == nil {
		return nil, 0, fmt.Errorf("客户端未连接")
	}

	deviceID := Init.Config.APP.Label
	if deviceID == "" {
		return nil, 0, fmt.Errorf("设备标识(APP.Label)未配置")
	}

	queryAPI := c.client.QueryAPI(c.org)
	measurement := fmt.Sprintf("device_%s", deviceID)
	pointIdStr := fmt.Sprintf("%d", pointId)

	// 构造 Flux 查询语句
	fluxQuery := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: time(v: "%s"), stop: time(v: "%s"))
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r.point_id == "%s")
			|> sort(columns: ["_time"], desc: true)
	`,
		c.bucket,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		measurement,
		pointIdStr,
	)

	// 执行查询
	result, err := queryAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("查询失败: %w", err)
	}
	defer result.Close()

	// 解析结果
	// InfluxDB v2 的 Flux 查询会为每个 field 返回一条记录，需要按时间戳分组
	// 使用毫秒级时间戳作为 key（截断到毫秒）
	typeTimeData := make(map[int64]map[string]interface{})
	var timeOrder []int64 // 保持时间顺序

	for result.Next() {
		record := result.Record()
		if record == nil {
			continue
		}

		// 截断到毫秒级精度
		timestampMs := record.Time().UnixMilli()
		field := record.Field()
		value := record.Value()

		// 初始化该时间点的数据
		if _, exists := typeTimeData[timestampMs]; !exists {
			typeTimeData[timestampMs] = map[string]interface{}{
				"Time": record.Time(),
			}
			timeOrder = append(timeOrder, timestampMs)
		}

		// 存储 field 值
		typeTimeData[timestampMs][field] = value
	}

	if err := result.Err(); err != nil {
		return nil, 0, fmt.Errorf("解析结果失败: %w", err)
	}

	// 转换为数组并按时间排序
	var allData []map[string]interface{}
	for _, timestamp := range timeOrder {
		item := typeTimeData[timestamp]

		// 提取 msg 和主要的 value 字段
		msg := ""
		if msgVal, ok := item["msg"]; ok {
			if s, ok := msgVal.(string); ok {
				msg = s
			}
		}

		// 找到第一个 value_* 字段作为主要值
		var mainValue interface{}
		var mainField string
		for key, val := range item {
			if len(key) >= 6 && key[:6] == "value_" {
				mainValue = val
				mainField = key
				break
			}
		}

		allData = append(allData, map[string]interface{}{
			"Time":  item["Time"],
			"Field": mainField,
			"Value": mainValue,
			"Msg":   msg,
		})
	}

	// 计算总数
	total := int64(len(allData))

	// 分页处理
	if pageSize > 0 {
		startIdx := int((page - 1) * pageSize)
		endIdx := startIdx + int(pageSize)

		if startIdx >= len(allData) {
			return []map[string]interface{}{}, total, nil
		}
		if endIdx > len(allData) {
			endIdx = len(allData)
		}

		allData = allData[startIdx:endIdx]
	}

	return allData, total, nil
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

// 写入缓存: key = point_id + 纳秒时间戳，防重复写入（PointId + Time 作为唯一索引）
// 优化：使用带过期时间的 map，定期清理旧数据防止内存泄漏
var (
	writeCache      sync.Map
	cacheCleanupMu  sync.Mutex
	lastCleanupTime time.Time
	cacheMaxAge     = 1 * time.Hour // 缓存最大保留时间
)

func init() {
	db_point.Update_Subscriber(a)
}

func a(value []fullConfig.Value_type) error {
	index := len(value)
	if index == 0 {
		return nil
	}

	value = append(value, fullConfig.Value_type{
		PointId: Init.Config.Influxdb.Write_Quantity_Id,
		Value:   index,
		Type:    "int",
		Msg:     "ok",
		Time:    time.Now(),
	})

	err := c.Write(value)
	if err != nil {
		log.Print(err.Error())
	}

	return nil
}

func New() (err error) {
	cfg := Init.Config.Influxdb
	if cfg.Url == "" || cfg.Token == "" || cfg.Org == "" || cfg.Bucket == "" {
		err = fmt.Errorf("InfluxDB 配置不完整，请检查配置文件")
		return
	}

	// 设置默认值
	writeTimeout := cfg.Write_Timeout
	if writeTimeout == 0 {
		writeTimeout = 5000 // 默认写入超时时间：5秒
	}

	bufferSize := cfg.BufferSize
	if bufferSize == 0 {
		bufferSize = 100 // 默认缓冲区大小：100条数据
	}

	flushInterval := cfg.FlushInterval
	if flushInterval == 0 {
		flushInterval = 2 * time.Second // 默认刷新间隔：2秒
	}

	c = Connect_struct{
		url:           cfg.Url,
		token:         cfg.Token,
		org:           cfg.Org,
		bucket:        cfg.Bucket,
		write_timeout: writeTimeout,
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
	}
	err = c.initInfluxDB()

	// 启动后台定时刷新协程
	c.startBackgroundFlush()

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
