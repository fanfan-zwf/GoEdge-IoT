/*
* 作用: 点位 Tag 路径树（tag_tree）懒加载 API
 */
package handler

import (
	"collector/internal/repo/mysql"

	"github.com/gin-gonic/gin"
)

// api_tag_tree_children 懒加载查询子节点
// POST /api/v1.0/tagtree/children  body: {"tag_path": "host1/机组1"}
//   - tag_path 为空：返回顶层根节点
//   - tag_path 非空：返回该路径的直接子节点
//
// 返回：{"子路径": [文件夹节点], "文件": [点位节点]}
func api_tag_tree_children(ctx *gin.Context) {
	var req struct {
		TagPath string `json:"tag_path"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	folders, files, err := mysql.Tag_Tree__Query_Children_ByPath(req.TagPath)
	if err != nil {
		ctx.Set("Response", []any{500, "查询子节点失败: " + err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", gin.H{
		"子路径": folders, // 文件夹节点（可展开）
		"文件":  files,   // 点位节点（可选中）
	}})
}

// api_tag_tree_rebuild 从 Point_Config 全量重建 tag_tree（增删改点位后调用一次即可）
// POST /api/v1.0/tagtree/rebuild
func api_tag_tree_rebuild(ctx *gin.Context) {
	if err := mysql.Tag_Tree__RebuildFromPointConfig(); err != nil {
		ctx.Set("Response", []any{500, "重建 tag_tree 失败: " + err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok"})
}

// api_tag_tree_points 查询某树节点路径下的全部点位（供“左树右表”联动过滤）
// POST /api/v1.0/tagtree/points  body: {"tag_path": "host1/机组1"}
//   - 点文件夹：返回其子树下所有点位；点单一点位：返回它自己
//
// 返回：直接为 Point_Config 记录列表（与 /config/point/query 同构），空路径返回空列表
func api_tag_tree_points(ctx *gin.Context) {
	var req struct {
		TagPath string `json:"tag_path"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Tag_Tree__Query_PointsUnderPath(req.TagPath)
	if err != nil {
		ctx.Set("Response", []any{500, "查询节点下点位失败: " + err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// tagtree_api_register 注册 tag_tree 相关路由
func tagtree_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/tagtree/children", api_tag_tree_children)
	r.POST("/api/v1.0/tagtree/points", api_tag_tree_points)
	r.POST("/api/v1.0/tagtree/rebuild", api_tag_tree_rebuild)
}
