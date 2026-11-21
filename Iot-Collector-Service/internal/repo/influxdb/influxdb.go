/*
* 日期: 2026.2.22 PM11:15
* 作者: 范范zwf
* 作用: influxdb
 */

package influxdb

import (
	"collector/internal/config"
	"collector/internal/model"
	"collector/internal/service/collector"
	"collector/pkg/conv"
	"sync"

	"context"
	"fmt"
	"log"
	"strings"
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
	writeAPI api.WriteAPI // 非阻塞写入API：客户端自带缓冲区，达到数量或时间阈值后批量发送
	queryAPI api.QueryAPI // 查询API（初始化时创建一次并复用）

	url    string // 地址
	token  string // 令牌
	org    string // 组织
	bucket string // 存储桶

	// 批量落盘参数（由 InfluxDB 客户端内置缓冲区管理：攒够数量 或 到时间 就发送一次）
	batch_size      uint          // 单批点数阈值（buffer_size）
	flush_interval  time.Duration // 缓冲区刷新间隔（flush_interval）
	request_timeout time.Duration // 单次 HTTP 请求超时（write_timeout）

	// 失败重传参数（InfluxDB 不可用时由客户端自动重传，业务层不需要自己缓存数据）
	max_retries    uint          // 写入失败最大重试次数
	retry_interval time.Duration // 重试等待间隔
	retry_buffer   uint          // 重试缓冲区最大点数（断网期间暂存待重传的数据）
	use_gzip       bool          // 写入请求是否 GZip 压缩

	// 「本批写入点数」统计点的 Tag（配置项 write_quantity_tag 归一化后的结果，初始化时生成）
	write_quantity_tag string

	// 写入与关闭的并发保护：关闭后不能再写入（向已关闭的通道发送数据会 panic）
	mu     sync.RWMutex
	closed bool
}

// 定义接口
type Connect_interface interface {
	Connect() error                            // 连接
	Close() error                              // 关闭连接
	Packet() error                             // 组包
	initInfluxDB() (err error)                 // 初始化InfluxDB客户端（程序启动时执行1次）
	Write(data []model.Value_type) (err error) // 批量写入函数

}

// 批量落盘/失败重传的默认参数（配置文件未提供时使用）
const (
	default_batch_size       uint          = 1000            // 默认单批点数
	default_flush_interval   time.Duration = 1 * time.Second // 默认刷新间隔
	default_request_timeout  time.Duration = 5 * time.Second // 默认HTTP请求超时
	default_max_retries      uint          = 3               // 默认重试次数
	default_retry_interval   time.Duration = 5 * time.Second // 默认重试间隔
	default_exponential_base uint          = 2               // 重试退避指数基数
)

// 序列标签（tag）名：历史数据全部按点位标识 Tag 检索
const Tag_Key = "tag"

// default_write_quantity_suffix 「本批写入点数」统计点的默认 Tag 尾段
// 该点不对应任何真实点位，只有一个 Tag 作为序列标识；可通过配置项 Influxdb.write_quantity_tag 改名字
const default_write_quantity_suffix = "write_quantity"

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

	// 默认值兜底
	if c.batch_size == 0 {
		c.batch_size = default_batch_size
	}
	if c.flush_interval <= 0 {
		c.flush_interval = default_flush_interval
	}
	if c.request_timeout <= 0 {
		c.request_timeout = default_request_timeout
	}
	if c.max_retries == 0 {
		c.max_retries = default_max_retries
	}
	if c.retry_interval <= 0 {
		c.retry_interval = default_retry_interval
	}
	if c.retry_buffer == 0 {
		// 重试缓冲区至少容纳「重试次数+1」批数据，保证短暂断网期间不丢数据
		c.retry_buffer = c.batch_size * (c.max_retries + 1)
	}

	// 统计点的 Tag 归一化：允许写成 /标识符 或 //设备标识//标识符，与点位 Tag 保持同一格式
	// 未配置或格式非法时回落到默认值 //<设备标识>//write_quantity
	if tag, tagErr := model.TagWithDeviceId(c.write_quantity_tag); tagErr == nil {
		c.write_quantity_tag = tag
	} else {
		if c.write_quantity_tag != "" {
			log.Printf("WARN InfluxDB 配置 write_quantity_tag(%q) 不符合点位 Tag 格式: %v", c.write_quantity_tag, tagErr)
		}
		if config.Config.APP.Label == "" {
			c.write_quantity_tag = ""
			log.Printf("WARN InfluxDB 统计点 Tag 无法生成：设备标识(APP.Label)未配置")
		} else {
			c.write_quantity_tag = "//" + config.Config.APP.Label + "//" + default_write_quantity_suffix
		}
	}

	options := influxdb2.DefaultOptions()
	// 批量落盘：攒够 batch_size 条 或 到 flush_interval 时间，客户端才发一次 HTTP 请求（客户端自带缓冲区，无需业务层再攒）
	options.SetBatchSize(c.batch_size)
	options.SetFlushInterval(uint(c.flush_interval.Milliseconds()))
	// 失败重传：写入失败的批次进入重试缓冲区，按指数退避策略自动重发
	options.SetMaxRetries(c.max_retries)
	options.SetRetryInterval(uint(c.retry_interval.Milliseconds()))
	options.SetRetryBufferLimit(c.retry_buffer)
	options.SetExponentialBase(default_exponential_base)
	// 单次 HTTP 请求超时，该参数单位为秒
	options.SetHTTPRequestTimeout(uint(c.request_timeout.Seconds()))
	// 边缘设备带宽有限，批量数据压缩后发送
	options.SetUseGZip(c.use_gzip)
	options.SetApplicationName("GoEdge-IoT Collector")

	// 创建客户端（全局复用，HTTP 长连接与连接池由客户端内部管理，不要每次写入都创建）
	c.client = influxdb2.NewClientWithOptions(c.url, c.token, options)
	// 非阻塞写入API：WritePoint 只把数据放入客户端缓冲区，发送与重传由客户端后台协程完成
	c.writeAPI = c.client.WriteAPI(c.org, c.bucket)
	// 查询API 同样只创建一次并复用
	c.queryAPI = c.client.QueryAPI(c.org)

	// 异步写入的错误只能从 Errors() 通道获取，必须持续读取，否则错误会被丢弃
	// 注意：Errors() 必须在首次写入之前调用，否则此前的错误不会被收集
	go c.watchWriteError(c.writeAPI.Errors())

	// 启动时探测一次服务可用性（最多等 2 秒，不拖慢启动）：
	// 探测失败不阻断程序，服务恢复前后的数据会由客户端重试队列自动重传，仅告警提示配置问题
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pingCancel()
	if ok, pingErr := c.client.Ping(pingCtx); pingErr != nil {
		log.Printf("WARN InfluxDB 连接探测失败(数据将进入重试缓冲区等待重传): %v", pingErr)
	} else if !ok {
		log.Printf("WARN InfluxDB 服务响应异常，请检查地址/令牌/组织配置")
	}

	return
}

// watchWriteError 消费客户端上报的异步写入错误（超出重试次数后该批次会被丢弃）
func (c *Connect_struct) watchWriteError(errCh <-chan error) {
	for err := range errCh {
		if err != nil {
			log.Printf("ERROR InfluxDB 写入失败: %v", err)
		}
	}
}

// Close 关闭连接
// 关闭前把缓冲区剩余数据落盘：client.Close() 内部会 Flush 并关闭写入API，错误监听协程随之退出
func (c *Connect_struct) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.client != nil {
		c.client.Close()
		c.client = nil
		c.writeAPI = nil
		c.queryAPI = nil
	}

	return nil
}

// Write 批量写入函数（非阻塞）
// 数据写入 InfluxDB 客户端自带的内存缓冲区，由客户端按「数量阈值 / 刷新间隔」批量落盘，
// 发送失败时由客户端按退避策略自动重传，因此业务层不再维护缓冲区和去重缓存
// 说明：InfluxDB 对「相同 measurement + tag + 时间戳」的写入是覆盖语义（天然幂等），重复写入不会产生多余数据
// 说明：服务不可用时批次会被放入重试队列后立即返回（不阻塞采集线程），超出重试缓冲区上限才丢弃最旧数据
func (c *Connect_struct) Write(data []model.Value_type) (err error) {
	if len(data) == 0 {
		return
	}

	// 读锁：确保写入期间不会被关闭（关闭后向已关闭通道发送数据会 panic）
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.writeAPI == nil {
		err = fmt.Errorf("InfluxDB 客户端未连接")
		return
	}

	// measurement 就是设备标识本身（config.yaml 中 APP.label），整批相同，提到循环外
	measurement := config.Config.APP.Label
	if measurement == "" {
		err = fmt.Errorf("设备标识(APP.Label)未配置")
		return
	}

	// 时序序列由 measurement + tag 唯一确定，Tag 为空的数据事后无法按点位检索，跳过并告警
	skipped := 0
	for i := range data {
		if data[i].Tag == "" {
			skipped++
			continue
		}
		c.writeAPI.WritePoint(New_Point(measurement, &data[i]))
	}
	if skipped > 0 {
		log.Printf("WARN InfluxDB 跳过 %d 条未携带 Tag 的数据（采集链路未补全点位标识）", skipped)
	}

	return
}

// Flush 强制刷新客户端缓冲区，把尚未发送的数据立即写入 InfluxDB
// 适用场景：希望「查询前/程序退出前」数据立即落盘（正常批量写入最多延迟 flush_interval）
func (c *Connect_struct) Flush() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.writeAPI == nil {
		return
	}
	c.writeAPI.Flush()
}

// New_Point 组装单个时序数据点
// 传入：measurement：设备标识（config.yaml 中的 APP.label）；v：点位值（v.Tag 作为序列标签）
// 返回：数据点，tags 为 tag（点位标识），fields 为 value_xxx（按实际类型分字段）+ msg（状态）
func New_Point(measurement string, v *model.Value_type) (point *write.Point) {
	fields := make(map[string]interface{}, 2)
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

	point = influxdb2.NewPoint(
		measurement, // measurement: 设备标识（APP.label）
		map[string]string{
			Tag_Key: v.Tag, // tag: 点位标识，历史查询按此过滤
		},
		fields,
		v.Time, // 时间戳
	)

	return
}

// 说明：旧版自定义的攒批缓冲区、后台定时刷新协程、sync.Map 去重缓存均已移除，
// 攒批落盘与失败重传由 InfluxDB 客户端内置缓冲区负责（见 initInfluxDB / Write）

// QueryByTagAndTime 按点位标识(Tag)和时间范围查询历史数据
func QueryByTagAndTime(tag string, startTime, endTime time.Time, page, pageSize uint) ([]map[string]interface{}, int64, error) {
	return c.queryByTagAndTime(tag, startTime, endTime, page, pageSize)
}

// flux_escape 转义 Flux 双引号字符串字面量中的特殊字符
// Tag 由用户配置，含引号/反斜杠时会破坏查询语句，这里做一次转义
func flux_escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// queryByTagAndTime 内部实现
func (c *Connect_struct) queryByTagAndTime(tag string, startTime, endTime time.Time, page, pageSize uint) ([]map[string]interface{}, int64, error) {
	if c.queryAPI == nil {
		return nil, 0, fmt.Errorf("InfluxDB 客户端未连接")
	}
	if tag == "" {
		return nil, 0, fmt.Errorf("点位标识(Tag)不能为空")
	}

	// measurement 就是设备标识本身（config.yaml 中 APP.label）
	measurement := config.Config.APP.Label
	if measurement == "" {
		return nil, 0, fmt.Errorf("设备标识(APP.Label)未配置")
	}

	// 查询API 已在初始化时创建，直接复用（每次重新创建会重复构建 HTTP 服务对象）
	queryAPI := c.queryAPI

	// 构造 Flux 查询语句（按 measurement + tag 定位序列）
	fluxQuery := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: time(v: "%s"), stop: time(v: "%s"))
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r.%s == "%s")
			|> sort(columns: ["_time"], desc: true)
	`,
		flux_escape(c.bucket),
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		flux_escape(measurement),
		Tag_Key,
		flux_escape(tag),
	)

	// 执行查询（带超时，避免服务不可用时请求长时间挂起）
	queryCtx, queryCancel := context.WithTimeout(context.Background(), c.request_timeout)
	defer queryCancel()
	result, err := queryAPI.Query(queryCtx, fluxQuery)
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
			"Tag":   tag,
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
	if c.queryAPI == nil {
		err = fmt.Errorf("influxdb客户端未连接")
		return
	}
	if len(scopes) == 0 {
		err = fmt.Errorf("读取范围数组为空")
		return
	}

	// 初始化返回结果集
	// 查询API 在初始化时创建并复用（v2必需）
	queryAPI := c.queryAPI

	// measurement 就是设备标识本身（config.yaml 中 APP.label），数据统一写入同一个 measurement
	measurement := config.Config.APP.Label
	if measurement == "" {
		err = fmt.Errorf("设备标识(APP.Label)未配置")
		return
	}

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
		// 校验支持的类型（与写入的 value_xxx 字段名一一对应）
		switch scope.Value_Type {
		case "int", "uint", "float", "string", "bool":
		default:
			err = fmt.Errorf("不支持的值类型：%s(Tag:%s),仅支持int/uint/float/string/bool", scope.Value_Type, scope.Tag)
			return
		}

		// 2. 字段名与写入逻辑完全匹配：value_ + 类型（value_int / value_float / value_string / value_bool）
		fieldName := "value_" + scope.Value_Type

		// 3. 构造Flux查询语句（measurement = 设备标识，序列由 tag 定位，值与状态分属两个 field）
		fluxQuery := fmt.Sprintf(`
			from(bucket: "%s")
				|> range(start: time(v: "%s"), stop: time(v: "%s"))
				|> filter(fn: (r) => r._measurement == "%s")
				|> filter(fn: (r) => r.%s == "%s")
				|> filter(fn: (r) => r._field == "%s" or r._field == "msg")
				|> sort(columns: ["_time"], desc: false) // 按时间升序排列
		`,
			flux_escape(c.bucket),                 // 存储桶名
			scope.Start_Time.Format(time.RFC3339), // 标准化时间格式
			scope.End_Time.Format(time.RFC3339),
			flux_escape(measurement), // 测量名 = 设备标识（与写入一致）
			Tag_Key,
			flux_escape(scope.Tag), // tag = 点位标识
			fieldName,              // 字段名 = value_+类型（如 value_int）
		)

		// 4. 执行查询（带超时，避免服务不可用时请求长时间挂起）
		var result *api.QueryTableResult
		queryCtx, queryCancel := context.WithTimeout(context.Background(), c.request_timeout)
		result, err = queryAPI.Query(queryCtx, fluxQuery)
		queryCancel()
		if err != nil {
			err = fmt.Errorf("查询失败(Tag:%s):%w", scope.Tag, err)
			return
		}

		// 5. 解析查询结果到结构体
		// 同一时间点的数据被拆成 value_xxx 与 msg 两条记录，按时间戳归并回一条
		dataByTime := make(map[int64]*Read_Scope_Value_Data_type)
		var timeOrder []int64 // 保持时间升序
		for result.Next() {
			record := result.Record()
			if record == nil {
				continue
			}

			key := record.Time().UnixNano()
			item, exists := dataByTime[key]
			if !exists {
				item = &Read_Scope_Value_Data_type{Time: record.Time()}
				dataByTime[key] = item
				timeOrder = append(timeOrder, key)
			}

			switch record.Field() {
			case fieldName:
				// 直接采用 InfluxDB 返回的原生类型（整数统一为 int64）
				item.Value = record.Value()
			case "msg":
				if msg, ok := record.Value().(string); ok {
					item.Msg = msg
				}
			}
		}

		// 6. 检查结果解析过程中的错误
		err = result.Err()
		// 循环内不能用 defer（否则要等整个函数返回才释放），这里显式关闭结果集
		result.Close()
		if err != nil {
			err = fmt.Errorf("解析查询结果失败(Tag:%s):%w", scope.Tag, err)
			return
		}

		for _, key := range timeOrder {
			scopeResult.Data = append(scopeResult.Data, *dataByTime[key])
		}

		// 7. 将当前点位的结果加入总结果集
		readResults = append(readResults, scopeResult)
	}

	return
}

var c Connect_struct

func init() {
	collector.Update_Subscriber(a)
}

// Close 关闭全局 InfluxDB 连接（程序退出时调用，会把缓冲区剩余数据先落盘）
func Close() error {
	return c.Close()
}

// Flush 强制刷新客户端缓冲区，把尚未发送的数据立即写入 InfluxDB
func Flush() {
	c.Flush()
}

// a 数据变化订阅回调：把本批变化的点位写入时序数据库
func a(value []model.Value_type) error {
	// 未启用 InfluxDB 时直接返回，避免每次数据变化都产生无效日志
	if !config.Config.Influxdb.Enable {
		return nil
	}

	index := len(value)
	if index == 0 {
		return nil
	}

	// 「本批写入点数」统计点：不是真实点位，没有采集链路带入的 Tag，用初始化时解析好的统计点 Tag
	// 统计点 Tag 不可用时只跳过它本身，不影响真实点位数据落盘
	if c.write_quantity_tag != "" {
		value = append(value, model.Value_type{
			Tag:   c.write_quantity_tag,
			Value: index,
			Type:  conv.TYPE_INT,
			Msg:   "ok",
			Time:  time.Now(),
		})
	}

	err := c.Write(value)
	if err != nil {
		log.Printf("ERROR InfluxDB 写入: %v", err)
	}

	return nil
}

func New() (err error) {
	cfg := config.Config.Influxdb
	if cfg.Url == "" || cfg.Token == "" || cfg.Org == "" || cfg.Bucket == "" {
		err = fmt.Errorf("InfluxDB 配置不完整，请检查配置文件")
		return
	}

	c = Connect_struct{
		url:                cfg.Url,
		token:              cfg.Token,
		org:                cfg.Org,
		bucket:             cfg.Bucket,
		batch_size:         cfg.BufferSize,
		flush_interval:     cfg.FlushInterval,
		request_timeout:    time.Duration(cfg.Write_Timeout) * time.Millisecond,
		max_retries:        cfg.Max_Retries,
		retry_interval:     cfg.Retry_Interval,
		retry_buffer:       cfg.Retry_Buffer,
		use_gzip:           cfg.Use_GZip,
		write_quantity_tag: cfg.Write_Quantity_Tag, // 原始配置值，格式归一化在 initInfluxDB 里完成
	}

	err = c.initInfluxDB()
	if err != nil {
		// 初始化失败时释放已创建的资源，避免留下半可用的客户端
		if closeErr := c.Close(); closeErr != nil {
			log.Printf("WARN InfluxDB 初始化失败后关闭连接异常: %v", closeErr)
		}
		return
	}

	log.Printf("INFO InfluxDB 初始化完成, 批量落盘: %d条 或 %v, 请求超时: %v, 失败重试: %d次(间隔%v, 重试缓冲%d条), gzip: %v, 统计点tag: %s",
		c.batch_size, c.flush_interval, c.request_timeout, c.max_retries, c.retry_interval, c.retry_buffer, c.use_gzip, c.write_quantity_tag)

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
