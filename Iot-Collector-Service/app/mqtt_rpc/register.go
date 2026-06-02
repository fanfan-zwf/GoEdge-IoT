package mqtt_rpc

import (
	"main/Init"
	"main/db/mysql"

	"fmt"
	"log"
	"reflect"
)

func app_restart(req []byte) (rep []byte, err error) {
	type Req struct {
	}

	return jsonWrap(req, func(r Req) (rep string, err error) {
		rep = "未开发！"
		err = fmt.Errorf("测试错误")
		return
	})
}

func collector_reload(req []byte) (rep []byte, err error) {
	type Req struct {
	}

	return jsonWrap(req, func(r Req) (rep string, err error) {

		rep = "ok"
		return
	})
}

func collector_synchronise_config(req []byte) (rep []byte, err error) {
	type Req struct {
	}

	return jsonWrap(req, func(r Req) (rep string, err error) {
		// 1. 获取采集器信息
		collectorList, err := Collector_Info__Search_Field("Uuid", 1, Init.Config.APP.Uuid)
		if err != nil {
			rep = err.Error()
			return
		}
		if len(collectorList) == 0 {
			rep = "未找到采集器信息"
			return
		}
		collectorID := collectorList[0].Id
		// ===================== 同步驱动配置 =====================
		// 1. 获取服务端驱动配置
		drive_List, err := Drive_Config__Query([]uint{collectorID}, []string{}, 0, 0)
		if err != nil {
			rep = err.Error()
			return
		}

		// 2. 构建 ID => index 映射（快速查找）
		driveIndexMap := make(map[uint]int, len(drive_List))
		for i, drive := range drive_List {
			driveIndexMap[drive.Id] = i
		}

		// 3. 定义最终要操作的数据
		var (
			drive_updateIDs []uint // 需要更新的 ID
			drive_delIDs    []uint // 需要删除的 ID
		)
		drive_runIndexs := make(map[int]bool) // 需要增加的下标

		// 4. 流式查询数据库数据 → 对比
		err = mysql.Drive_Config__Query_Callback([]uint{}, []string{}, 0, 0, func(dbDrive mysql.Drive_Config_type) {
			// 从服务端配置中查找
			idx, exists := driveIndexMap[dbDrive.Id]
			drive_runIndexs[idx] = true
			if !exists {
				// 数据库有，但服务端没有 → 删除
				drive_delIDs = append(drive_delIDs, dbDrive.Id)
				return
			}

			// 安全判断（你原来的逻辑写反了）
			if idx < 0 || idx >= len(drive_List) {
				err = fmt.Errorf("索引越界 idx=%d, len=%d", idx, len(drive_List))
				rep = err.Error()
				return
			}

			// 对比配置是否一致
			svcDrive := drive_List[idx]
			if !reflect.DeepEqual(svcDrive, dbDrive) {
				drive_updateIDs = append(drive_updateIDs, svcDrive.Id)
			}
		})
		if err != nil {
			rep = err.Error()
			return
		}

		// 5. 剩下的都是：服务端有、数据库没有 → 新增
		var drive_addList []mysql.Drive_Config_Add_type
		for _, idx := range driveIndexMap {
			_, ok := drive_runIndexs[idx]
			if ok {
				continue
			}

			if idx < 0 || idx >= len(drive_List) {
				err = fmt.Errorf("新增索引越界 idx=%d, len=%d", idx, len(drive_List))
				rep = err.Error()
				return
			}
			d := drive_List[idx]
			drive_addList = append(drive_addList, mysql.Drive_Config_Add_type{
				Id:           d.Id,
				Name:         d.Name,
				Config:       d.Config,
				Type:         d.Type,
				Collector_Id: collectorID,
			})
		}

		// 6. 构建更新列表
		var drive_updateList []mysql.Drive_Config_Synchronization_type
		for _, id := range drive_updateIDs {
			idx, ok := driveIndexMap[id] // 这里用原来的 map 最安全
			if !ok {
				continue
			}
			if idx < 0 || idx >= len(drive_List) {
				continue
			}
			d := drive_List[idx]
			drive_updateList = append(drive_updateList, mysql.Drive_Config_Synchronization_type{
				Drive_Config_Add_type: mysql.Drive_Config_Add_type{
					Id:            d.Id,
					Type:          d.Type,
					Name:          d.Name,
					Config:        d.Config,
					Creation_Time: d.Creation_Time,
				},
				Points_Length: d.Points_Length, // 点位数量
			})
		}

		// 最终执行 =====================
		// 更新
		if len(drive_updateList) != 0 {
			err = mysql.Drive_Config__Synchronization(drive_updateList...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		// 删除
		if len(drive_delIDs) != 0 {
			err = mysql.Drive_Config__Del(drive_delIDs...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		// 新增
		if len(drive_addList) != 0 {
			err = mysql.Drive_Config__Add(drive_addList...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		// ===================== 同步点位配置 =====================
		// 1. 获取服务端驱动配置
		Points_List, err := Points_Config__Query([]uint{collectorID}, []uint{}, 0, 0)
		if err != nil {
			rep = err.Error()
			return
		}

		// 2. 构建 ID => index 映射（快速查找）
		PointsIndexMap := make(map[uint]int, len(Points_List))
		for i, Points := range Points_List {
			PointsIndexMap[Points.Id] = i
		}

		// 3. 定义最终要操作的数据
		var (
			Points_updateIDs []uint // 需要更新的 ID
			Points_delIDs    []uint // 需要删除的 ID
		)
		Points_runIndexs := make(map[int]bool) // 需要增加的下标

		// 4. 流式查询数据库数据 → 对比
		err = mysql.Points_Config__Query_Callback([]uint{}, 0, 0, func(dbPoints mysql.Points_Config_type) {
			// 从服务端配置中查找
			idx, exists := PointsIndexMap[dbPoints.Id]
			Points_runIndexs[idx] = true
			if !exists {
				// 数据库有，但服务端没有 → 删除
				Points_delIDs = append(Points_delIDs, dbPoints.Id)
				return
			}

			// 安全判断（你原来的逻辑写反了）
			if idx < 0 || idx >= len(Points_List) {
				err = fmt.Errorf("索引越界 idx=%d, len=%d", idx, len(Points_List))
				rep = err.Error()
				return
			}

			// 对比配置是否一致
			svcPoints := Points_List[idx]
			if !reflect.DeepEqual(svcPoints, dbPoints) {
				Points_updateIDs = append(Points_updateIDs, svcPoints.Id)
			}
		})
		if err != nil {
			rep = err.Error()
			return
		}

		// 5. 剩下的都是：服务端有、数据库没有 → 新增
		var Points_addList []mysql.Points_Config_Add_type
		for _, idx := range PointsIndexMap {
			_, ok := Points_runIndexs[idx]
			if ok {
				continue
			}

			if idx < 0 || idx >= len(Points_List) {
				err = fmt.Errorf("新增索引越界 idx=%d, len=%d", idx, len(Points_List))
				rep = err.Error()
				return
			}
			d := Points_List[idx]
			Points_addList = append(Points_addList, mysql.Points_Config_Add_type{
				Id:          d.Id,          // 点位id
				Drive_Id:    d.Drive.Id,    // 驱动id唯一标识符
				Tag:         d.Tag,         // 点位标识
				Description: d.Description, // 点位描述
				RW_Cancel:   d.RW_Cancel,   // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
				Value_Type:  d.Value_Type,  // 输出类型
				Config:      d.Config,
			})
		}

		// 6. 构建更新列表
		var Points_updateList []mysql.Points_Config_Synchronization_type
		for _, id := range Points_updateIDs {
			idx, ok := PointsIndexMap[id] // 这里用原来的 map 最安全
			if !ok {
				continue
			}
			if idx < 0 || idx >= len(Points_List) {
				continue
			}
			d := Points_List[idx]
			Points_updateList = append(Points_updateList, mysql.Points_Config_Synchronization_type{
				Points_Config_Add_type: mysql.Points_Config_Add_type{
					Id:          d.Id,          // 点位id
					Drive_Id:    d.Drive.Id,    // 驱动id唯一标识符
					Tag:         d.Tag,         // 点位标识
					Description: d.Description, // 点位描述
					RW_Cancel:   d.RW_Cancel,   // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
					Value_Type:  d.Value_Type,  // 输出类型
					Config:      d.Config,
				},
				Creation_Time: d.Creation_Time, // 点位数量
			})
		}

		// 最终执行 =====================
		// 更新
		if len(Points_updateList) != 0 {
			err = mysql.Points_Config__Synchronization(Points_updateList...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		// 删除
		if len(Points_delIDs) != 0 {
			err = mysql.Points_Config__Del(Points_delIDs...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		// 新增
		if len(Points_addList) != 0 {
			err = mysql.Points_Config__Add(Points_addList...)
			if err != nil {
				rep = err.Error()
				return
			}
		}

		rep = "ok"
		return
	})
}

func register() {
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/Order/App/Restart", app_restart)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/Order/Collector/Config", collector_synchronise_config)
	M.Register(Init.Config.Mqtt_Rpc.Broker, "/Order/Collector/Reload", collector_reload)
}
func Collector_Info__Count(page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		Page     uint
		PageSize uint
	}
	err = jsonCall(Req{Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Collector/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

func Collector_Info__Query(page uint, pageSize uint) (resp []mysql.Collector_Info_type, err error) {
	type Req struct {
		Page     uint
		PageSize uint
	}

	err = jsonCall(Req{Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Collector/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

// 采集-》搜索
// 传递：field quantity 数量，vague 模糊搜索字符串
// 返回：configs 配置，err 错误
func Collector_Info__Search_Field(field string, quantity uint, vague string) (resp []mysql.Collector_Info_type, err error) {
	type Req struct {
		Field    string
		Quantity uint
		Vague    string
	}

	err = jsonCall(Req{Field: field, Quantity: quantity, Vague: vague}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Collector/Search/Field",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

func Drive_Config__Count(collectorId []uint, driveType []string, page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		CollectorId []uint
		DriveType   []string
		Page        uint
		PageSize    uint
	}
	err = jsonCall(Req{CollectorId: collectorId, DriveType: driveType, Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}

func Drive_Config__Query(collectorId []uint, driveType []string, page uint, pageSize uint) (resp []mysql.Drive_Config_type, err error) {
	type Req struct {
		CollectorId []uint
		DriveType   []string
		Page        uint
		PageSize    uint
	}

	err = jsonCall(Req{CollectorId: collectorId, DriveType: driveType, Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR RPC调用失败：%v", err)
		log.Print(err)
	}
	return
}
func Points_Config__Count(Collector_Id []uint, driveid []uint, page uint, pageSize uint) (resp uint, err error) {
	type Req struct {
		Collector_Id []uint
		Driveid      []uint
		Page         uint
		PageSize     uint
	}
	err = jsonCall(Req{Collector_Id: Collector_Id, Driveid: driveid, Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Drive/Count",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR ：%v", err)
		log.Print(err)
	}
	return
}

func Points_Config__Query(Collector_Id []uint, driveid []uint, page uint, pageSize uint) (resp []mysql.Points_Config_type, err error) {
	type Req struct {
		Collector_Id []uint
		Driveid      []uint
		Page         uint
		PageSize     uint
	}

	err = jsonCall(Req{Collector_Id: Collector_Id, Driveid: driveid, Page: page, PageSize: pageSize}, &resp,
		Init.Config.Mqtt_Rpc.Broker,
		Init.Config.Mqtt_Rpc.ConfigServiceTopic,
		"/V1.0/Get/Points/Query",
		Init.Config.Mqtt_Rpc.BusinessTimeout,
	)
	if err != nil {
		err = fmt.Errorf("ERROR %v", err)
		log.Print(err)
	}
	return
}
