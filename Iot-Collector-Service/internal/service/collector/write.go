/*
* 日期: 2026.11.19
* 作者: 范范zwf
* 作用: 写值标准入口。把原本在 HTTP handler 与内部 MQTT 各自重复的写值流程收敛到本包，分两段：
*
*       1) PrepareWrite：把请求入参（Tag / pointId 定位 + 原始 JSON 值 + 类型名）解析定位、
*          精确转换成统一的 []model.Value_type 数组；
*       2) WriteValues：接收 []model.Value_type 数组，按驱动分组并逐驱动下发（只读校验在
*          driver.DriveWrite 入口统一执行）。
*
*       API 与 MQTT 两条通路都以「[]model.Value_type 数组」作为标准入参调用 WriteValues，
*       实现下发逻辑的完全复用。
*
*       设计约定：
*       - 本层不依赖 gin / 网络上下文；写值审计涉及的账号 / IP 属于请求上下文，由调用方补全落库；
*       - 错误用哨兵（ErrWriteNotFound / ErrWriteInvalid）标注语义，由上层映射为业务码；
*         下发阶段的只读拒绝以 driver.ErrPointNotWritable 原样透传，供上层区分“权限/参数非法”与
*         “设备写入失败”。未标注哨兵的按基础设施错误处理（DB 查询失败等）。
 */

package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"collector/internal/model"
	"collector/internal/repo/mysql"
	"collector/internal/service/driver"
	"collector/pkg/conv"
)

// 写值入口错误语义（供上层映射 HTTP 码：NotFound→404，Invalid/不可写→417，其余→500）
var (
	// ErrWriteNotFound 点位定位失败：Tag 反查不到 id，或 id 无对应驱动关系
	ErrWriteNotFound = errors.New("点位不存在")
	// ErrWriteInvalid 请求非法：Tag 与 pointId 同时为空、类型不存在、值解析失败
	ErrWriteInvalid = errors.New("写值参数错误")
)

// WriteInput 单条写值入参。Tag 与 PointId 二者取一（Tag 优先），Value 为值的原始 JSON 报文，
// Type 为值类型名（bool/int8/.../string，与 conv.ValueTypeCode 的键一致）。
type WriteInput struct {
	Tag     string          // 点位标识（优先）
	PointId uint            // 点位ID（未传 Tag 时按 id 定位）
	Value   json.RawMessage // 写入值的原始 JSON
	Type    string          // 值类型名
}

// writeValueTypeMap 值类型名 → reflect.Type，用于把 json 解出的通用值精确转换成目标类型。
var writeValueTypeMap = map[string]reflect.Type{
	"bool":    reflect.TypeOf(bool(false)),
	"int8":    reflect.TypeOf(int8(0)),
	"uint8":   reflect.TypeOf(uint8(0)),
	"int16":   reflect.TypeOf(int16(0)),
	"uint16":  reflect.TypeOf(uint16(0)),
	"int32":   reflect.TypeOf(int32(0)),
	"uint32":  reflect.TypeOf(uint32(0)),
	"int64":   reflect.TypeOf(int64(0)),
	"uint64":  reflect.TypeOf(uint64(0)),
	"int":     reflect.TypeOf(int(0)),
	"uint":    reflect.TypeOf(uint(0)),
	"float32": reflect.TypeOf(float32(0)),
	"float64": reflect.TypeOf(float64(0)),
	"float":   reflect.TypeOf(float64(0)),
	"string":  reflect.TypeOf(""),
}

// PrepareWrite 写值解析入口：把请求入参定位点位、精确转换值，产出统一的 []model.Value_type。
// 返回数组与入参等长、顺序一致（调用方可按下标回填原始 JSON 做审计）。
// 定位与转换阶段任一错误即整批中止，返回携带语义哨兵的错误。
// 传入：inputs — 待写入的点位值列表（Tag 或 PointId 定位，可混用）
// 返回：[]model.Value_type（PointId/Tag/Value/Type/Time 均已就绪）、error
func PrepareWrite(inputs []WriteInput) ([]model.Value_type, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// 1. Tag 优先：带了 Tag 的项先反查成点位 id（内部按 id 找驱动关系）
	tags := make([]string, 0, len(inputs))
	for _, it := range inputs {
		if it.Tag != "" {
			tags = append(tags, it.Tag)
		}
	}
	tagIdMap, missing, err := writeIdsByTags(tags)
	if err != nil {
		return nil, fmt.Errorf("查询点位Tag失败: %w", err)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrWriteNotFound, strings.Join(missing, ", "))
	}

	// 2. 逐项定好 id 与 Tag：只传了 pointId 的按 id 反查 Tag，
	//    保证写进驱动的命令与采集回来的历史数据用同一套点位标识
	ids := make([]uint, len(inputs))
	itemTags := make([]string, len(inputs))
	idOnly := make([]uint, 0, len(inputs))
	for i, it := range inputs {
		ids[i] = it.PointId
		if it.Tag != "" {
			ids[i] = tagIdMap[it.Tag] // Tag 优先：与 pointId 同时给出时以 Tag 为准
		}
		if ids[i] == 0 {
			return nil, fmt.Errorf("%w: 第 %d 条点位标识(tag)与点位ID(pointId)不能同时为空", ErrWriteInvalid, i+1)
		}
		itemTags[i] = it.Tag
		if itemTags[i] == "" {
			idOnly = append(idOnly, ids[i])
		}
	}
	if len(idOnly) > 0 {
		idTagMap, err := writeTagsByIds(idOnly)
		if err != nil {
			return nil, fmt.Errorf("查询点位Tag失败: %w", err)
		}
		for i := range itemTags {
			if itemTags[i] == "" {
				itemTags[i] = idTagMap[ids[i]]
			}
		}
	}

	// 3. 精确转换值，产出与入参等长、顺序一致的待写值数组
	values := make([]model.Value_type, 0, len(inputs))
	for i, it := range inputs {
		var realValue any
		if err = json.Unmarshal(it.Value, &realValue); err != nil {
			return nil, fmt.Errorf("%w: 第 %d 条值解析失败: %s", ErrWriteInvalid, i+1, err.Error())
		}

		t, ok := writeValueTypeMap[it.Type]
		if !ok {
			return nil, fmt.Errorf("%w: 第 %d 条不存在的类型: %s", ErrWriteInvalid, i+1, it.Type)
		}
		realValue = reflect.ValueOf(realValue).Convert(t).Interface()

		values = append(values, model.Value_type{
			PointId: ids[i],
			Tag:     itemTags[i], // 点位标识：下游（InfluxDB 历史数据）按此建序列
			Value:   realValue,
			Type:    conv.ValueTypeCode[it.Type],
			Time:    time.Now(),
		})
	}

	return values, nil
}

// WriteValues 写值下发入口（标准入参为 []model.Value_type，API 与 MQTT 共用）：
// 按点位所属驱动分组后逐驱动调用 driver.DriveWrite；只读属性校验在 DriveWrite 入口统一执行。
// 传入：values — 已定位点位、已转换类型的待写值数组
// 返回：error（只读拒绝透传 driver.ErrPointNotWritable；定位/驱动缺失带 ErrWriteNotFound；其余为下发失败）
func WriteValues(values []model.Value_type) error {
	if len(values) == 0 {
		return nil
	}

	// 1. 查询各点位所属驱动
	ids := make([]uint, 0, len(values))
	for _, v := range values {
		ids = append(ids, v.PointId)
	}
	driveIdList, err := mysql.Point_Config__Query__DriveIds(ids...)
	if err != nil {
		return fmt.Errorf("查询点位驱动关系失败: %w", err)
	}
	pointDriveMap := make(map[uint]uint, len(driveIdList))
	for _, d := range driveIdList {
		pointDriveMap[d.Id] = d.Drive_Id
	}

	// 2. 按 Drive_Id 分组
	driveGroups := make(map[uint][]model.Value_type)
	for _, v := range values {
		driveId, exists := pointDriveMap[v.PointId]
		if !exists {
			return fmt.Errorf("%w: id=%d tag=%s", ErrWriteNotFound, v.PointId, v.Tag)
		}
		driveGroups[driveId] = append(driveGroups[driveId], v)
	}

	// 3. 逐驱动下发
	for driveId, vals := range driveGroups {
		if err := driver.DriveWrite(driveId, vals); err != nil {
			// 只读/禁止拒绝原样透传，保留 ErrPointNotWritable 供上层映射为客户端错误码
			if errors.Is(err, driver.ErrPointNotWritable) {
				return err
			}
			return fmt.Errorf("驱动 id=%d 写入失败: %w", driveId, err)
		}
	}

	return nil
}

// writeIdsByTags 按 Tag 批量反查点位 id。
// 返回：idMap（Tag -> id，只含库里存在的）、missing（库里查不到的 Tag）
func writeIdsByTags(tags []string) (idMap map[string]uint, missing []string, err error) {
	idMap = make(map[string]uint, len(tags))
	if len(tags) == 0 {
		return
	}

	list, err := mysql.Point_Config__Query__IdTag_ByTags(tags...)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range list {
		idMap[item.Tag] = item.Id
	}

	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, ok := idMap[tag]; !ok {
			missing = append(missing, tag)
		}
	}
	return idMap, missing, nil
}

// writeTagsByIds 按点位 id 批量查询 Tag，库里不存在或 Tag 为空的 id 不出现在结果中
func writeTagsByIds(ids []uint) (tagMap map[uint]string, err error) {
	tagMap = make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return
	}

	list, err := mysql.Point_Config__Query__IdTag_ByIds(ids...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.Tag != "" {
			tagMap[item.Id] = item.Tag
		}
	}
	return tagMap, nil
}
