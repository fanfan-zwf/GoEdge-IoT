/*
* 日期: 2026.2.20 PM7:52
* 作者: 范范zwf
* 作用: 实时数据库——基于redis
 */

package db_point

import (
	"fmt"
	"log"
	"sync"
)

/*
******************写入******************
 */
type Write_value_type struct {
	Points_Id  uint   // 点位id
	Value_Type string // 值类型
	Time       string
	Value      any
}

type Write_value_func_type func(Write_value_type) (exist bool, err error)

var (
	Write_value    []*Write_value_func_type
	Write_value_mu sync.Mutex
)

// 变化更新 发布 发送
func Write_value_Publisher(value Write_value_type) error {
	var ok bool

	for i, v := range Write_value {
		if v == nil {
			Write_value_mu.Lock() // 这里要阻塞运行，防止同步删除
			Write_value = append(Write_value[:i], Write_value[i+1:]...)
			log.Printf("WARNING 关闭一个 变化更新 的订阅者 %d", i)
			Write_value_mu.Unlock()
			continue
		}
		exist, err := (*v)(value)
		if exist && err == nil {
			ok = true
			break
		}
		if err != nil {
			if err == Err_Publisher_Close {
				Write_value[i] = nil
			}
			log.Printf("ERROR 值写入错误: %v", err)
		}
	}

	if ok {
		log.Printf("INFO 写值->  Point:%d, Value:%v,", value.Points_Id, value.Value)
		return nil
	}

	return fmt.Errorf("找不到%d点位", value.Points_Id)
}

// 变化更新 订阅 接收
func Write_value_Subscriber(value Write_value_func_type) error {
	Write_value = append(Write_value, &value)
	return nil
}
