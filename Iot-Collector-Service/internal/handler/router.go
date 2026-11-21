/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: Web API 路由注册入口
 */
package handler

import (
	"github.com/gin-gonic/gin"
)

func gui_api(r *gin.Engine) {
	// 鉴权相关 API（登录/刷新/内部同步，这三条由中间件按路径豁免；其余业务接口共用鉴权中间件）
	auth_api_register(r)

	// 点位读写 + WebSocket + 报警状态 + 历史数据查询
	point_api_register(r)

	// 配置管理 API
	config_api_register(r)

	// 登录用户（user_account）管理 API（受鉴权中间件保护）
	user_api_register(r)

	// AK/SK 接口账号（api_key）管理 API（受鉴权中间件保护）
	apikey_api_register(r)

	// RBAC 权限管理 API（角色/权限字典/绑定，受鉴权 + 权限中间件保护）
	rbac_admin_api_register(r)

	// 操作日志查询/清理 API（log_*_config / log_*_login / log_*_write）
	log_api_register(r)

	// 系统配置管理 API（config.yaml 读写）
	system_config_api_register(r)

	// 系统性能信息 API（CPU、内存、硬盘等）
	system_info_api_register(r)

	// 点位 Tag 路径树（tag_tree）懒加载 API
	tagtree_api_register(r)

	// 接口说明页（浏览器直接打开 API 根路径即可查看全部已注册接口）。
	// 清单在请求时由 r.Routes() 生成，与注册顺序无关，放在最后只是为了语义上“先业务后元信息”。
	docs_api_register(r)
}
