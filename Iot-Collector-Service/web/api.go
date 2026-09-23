/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: Web API 路由注册入口
 */
package web

import (
	"github.com/gin-gonic/gin"
)

func gui_api(r *gin.Engine) {
	// 点位读写 + WebSocket + 报警状态 + 历史数据查询
	point_api_register(r)

	// 配置管理 API
	config_api_register(r)
}
