/*
* 作用: tag_tree 点位 Tag 路径树（懒加载）的构建与查询
* 说明: 从 Point_Config.Tag 解析层级路径建树。Tag 支持两种形态：
*   1. //[host]//机组1/泵A/温度  —— host 作为树的顶层文件夹，其下是变长路径
*   2. /机组1/泵A/温度            —— 无 host，直接从 机组1 起建树
* 规则: Tag 永不以 / 结尾，故路径末段一定是"点位本身"；
*   同一节点可能既是点位（Point_Id 非空）又是文件夹（Has_Children=1），二者正交。
* 寻址: Path 为全局唯一完整路径（如 /host1/机组1/泵A），API 仅凭 tag_path 即可定位。
 */

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// tag_tree 节点类型
const (
	Tag_Tree_Node_Folder = 1 // 纯文件夹
	Tag_Tree_Node_Point  = 2 // 纯点位
	Tag_Tree_Node_Both   = 3 // 既是点位又是文件夹
)

// tag_tree 查询返回结构体
type Tag_Tree_node_type struct {
	Id           uint   `json:"id"`
	Host         string `json:"host"`
	Parent_Id    uint   `json:"parentId"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	Level        int    `json:"level"`
	Node_Type    int    `json:"nodeType"`
	Point_Id     uint   `json:"pointId"`     // 0=无（纯文件夹）；非0=可选中点位，指向 Point_Config.Id
	Has_Children int    `json:"hasChildren"` // 1=可展开
}

// 匹配 //[host]//剩余路径；host 不含 /，剩余部分至少一段
var reTagHost = regexp.MustCompile(`^//([^/]+)//(.+)$`)

// tagPath_Parse 解析 Tag -> host + 树路径段
// 传递：tag —— Point_Config.Tag 原文
// 返回：host（纯路径形态为 ""）、segs（树层级段）、ok（是否解析成功）
func tagPath_Parse(tag string) (host string, segs []string, ok bool) {
	if m := reTagHost.FindStringSubmatch(tag); m != nil {
		host = m[1]
		segs = strings.Split(strings.Trim(m[2], "/"), "/")
	} else if strings.HasPrefix(tag, "/") {
		segs = strings.Split(strings.Trim(tag, "/"), "/")
	} else {
		return "", nil, false
	}
	if len(segs) == 0 || segs[0] == "" {
		return "", nil, false
	}
	return host, segs, true
}

// Tag_Tree__RebuildFromPointConfig 从 Point_Config 全量重建 tag_tree（事务、幂等）
func Tag_Tree__RebuildFromPointConfig() (err error) {
	points, err := Point_Config__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("读取 Point_Config 失败: %w", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	if _, err = tx.Exec("DELETE FROM `tag_tree`"); err != nil {
		err = fmt.Errorf("清空 tag_tree 失败: %w", err)
		return
	}

	// 借 uk_path 唯一键做 upsert：命中已存在节点返回旧 Id；
	// Point_Id 用 COALESCE 保护，文件夹段（NULL）不会清空已写入的点位 Id
	upsert := `INSERT INTO tag_tree (Host, Parent_Id, Name, Path, Level, Point_Id, Creation_Time)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE Id = LAST_INSERT_ID(Id), Point_Id = COALESCE(VALUES(Point_Id), Point_Id)`

	getNode := func(host string, parentId uint, parentPath, name string, level int, pointId any) (uint, error) {
		path := parentPath + "/" + name // 全局唯一完整路径（host 作为顶层段）
		res, e := tx.Exec(upsert, host, parentId, name, path, level, pointId, time.Now())
		if e != nil {
			return 0, fmt.Errorf("写入树节点[%s]失败: %w", path, e)
		}
		id, e := res.LastInsertId()
		if e != nil {
			return 0, fmt.Errorf("获取树节点[%s]Id失败: %w", path, e)
		}
		return uint(id), nil
	}

	for _, p := range points {
		host, segs, ok := tagPath_Parse(p.Tag)
		if !ok {
			log.Printf("WARN 点位【%s】 Tag 无法解析路径, 跳过建树", p.Tag)
			continue
		}
		// host 作为顶层文件夹；无 host（纯路径形态）则直接从首段建起
		treeSegs := segs
		if host != "" {
			treeSegs = append([]string{host}, segs...)
		}
		var parentId uint
		parentPath := ""
		for i, seg := range treeSegs {
			var pointId any // nil => 该段作为文件夹存在
			if i == len(treeSegs)-1 {
				pointId = p.Id // 末段=点位本身
			}
			id, e := getNode(host, parentId, parentPath, seg, i+1, pointId)
			if e != nil {
				err = e
				return
			}
			parentId = id
			parentPath = parentPath + "/" + seg
		}
	}

	// 回填 Has_Children：凡被引用为父节点的置 1
	if _, err = tx.Exec(`UPDATE tag_tree p
		JOIN (SELECT DISTINCT Parent_Id FROM tag_tree WHERE Parent_Id <> 0) c ON p.Id = c.Parent_Id
		SET p.Has_Children = 1`); err != nil {
		err = fmt.Errorf("回填 Has_Children 失败: %w", err)
		return
	}

	// 派生 Node_Type
	if _, err = tx.Exec(`UPDATE tag_tree SET Node_Type = CASE
		WHEN Point_Id IS NOT NULL AND Has_Children = 1 THEN 3
		WHEN Point_Id IS NOT NULL THEN 2
		ELSE 1 END`); err != nil {
		err = fmt.Errorf("派生 Node_Type 失败: %w", err)
		return
	}
	return
}

// Tag_Tree__Query_Children_ByPath 按 tag_path 懒加载查询其直接子节点
// 传递：tagPath —— 当前节点完整路径（如 host1/机组1，前导/有无均可）；空串表示加载顶层根节点
// 返回：folders —— 子路径（Has_Children=1 的文件夹节点）；files —— 文件（Point_Id!=0 的点位节点）
// 说明：“既是点位又是文件夹”的节点（Node_Type=3）会同时出现在两个列表中
func Tag_Tree__Query_Children_ByPath(tagPath string) (folders, files []Tag_Tree_node_type, err error) {
	folders = []Tag_Tree_node_type{}
	files = []Tag_Tree_node_type{}

	tagPath = strings.Trim(tagPath, "/")
	const cols = `Id, Host, Parent_Id, Name, Path, Level, Node_Type, IFNULL(Point_Id, 0), Has_Children`

	var childrenSQL string
	var args []any
	if tagPath == "" {
		childrenSQL = `SELECT ` + cols + ` FROM tag_tree WHERE Parent_Id = 0 ORDER BY Node_Type, Name`
	} else {
		childrenSQL = `SELECT ` + cols + ` FROM tag_tree
			WHERE Parent_Id = (SELECT Id FROM tag_tree WHERE Path = ?) ORDER BY Node_Type, Name`
		args = append(args, "/"+tagPath)
	}

	rows, err := DB.Query(childrenSQL, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("查询子节点失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Tag_Tree_node_type
		if err = rows.Scan(&n.Id, &n.Host, &n.Parent_Id, &n.Name, &n.Path, &n.Level, &n.Node_Type, &n.Point_Id, &n.Has_Children); err != nil {
			return nil, nil, fmt.Errorf("扫描子节点失败: %w", err)
		}
		if n.Point_Id != 0 {
			files = append(files, n) // 点位（文件）
		}
		if n.Has_Children == 1 {
			folders = append(folders, n) // 文件夹（子路径）
		}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("遍历子节点结果集失败: %w", err)
	}
	return folders, files, nil
}

// Tag_Tree__Query_PointsUnderPath 查询某树节点路径下的全部点位（含自身为点位的情况）
// 传递：tagPath —— 节点完整路径（如 host1/机组1，前导/有无均可）；空串返回空
// 返回：该子树内所有 Point_Id 对应的 Point_Config 记录，错误
// 说明：以 Parent_Id 沿 tag_tree 递归下钻定位子树节点，再用其 Point_Id 关联 Point_Config；
//
//	点文件夹得到其下全部点位，点单一点位得到它自己（Point_Id 精确关联，不用 LIKE，避免通配符误匹配）
func Tag_Tree__Query_PointsUnderPath(tagPath string) (configs []Point_Config_Query_type, err error) {
	tagPath = strings.Trim(tagPath, "/")
	if tagPath == "" {
		return // 空路径不取，由前端走常规分页查询
	}

	// 递归 CTE：从目标路径节点出发，沿 Parent_Id 收集整棵子树，再按 Point_Id 关联点位配置
	const q = `WITH RECURSIVE sub AS (
		SELECT Id, Point_Id, Path FROM tag_tree WHERE Path = ?
		UNION ALL
		SELECT t.Id, t.Point_Id, t.Path FROM tag_tree t JOIN sub s ON t.Parent_Id = s.Id
	)
	SELECT p.Id, p.Drive_Id, p.Tag, p.Description, p.Config, p.Conversion, p.RW_Cancel, p.Value_Type, p.Creation_Time
	FROM Point_Config p JOIN sub s ON s.Point_Id = p.Id
	WHERE s.Point_Id <> 0
	ORDER BY s.Path`

	rows, err := DB.Query(q, "/"+tagPath)
	if err != nil {
		return nil, fmt.Errorf("查询节点下点位失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Point_Config_Query_type
		var desc sql.NullString
		if err = rows.Scan(&cfg.Id, &cfg.Drive_Id, &cfg.Tag, &desc, &cfg.Config, &cfg.Conversion, &cfg.RW_Cancel, &cfg.Value_Type, &cfg.Creation_Time); err != nil {
			return nil, fmt.Errorf("扫描节点下点位失败: %w", err)
		}
		cfg.Description = desc.String // NULL 转空串，与常规点位查询保持一致
		configs = append(configs, cfg)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历节点下点位结果集失败: %w", err)
	}
	return configs, nil
}
