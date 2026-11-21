/*
***************统一鉴权分表：user_account（网页登录）+ api_key（AK/SK 接口）***************
 */

package mysql

import (
	"collector/pkg/conv"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// ============================================================
// user_account —— 网页登录用户（账号密码，供 Bearer token 通路使用）
// ============================================================

// user_account 令牌时效默认值（以 time.Duration 存储为 duration 字符串，如 "15m0s"）
const (
	Default_User_Access_TTL  = conv.Duration(15 * time.Minute)   // 900s  = "15m0s"
	Default_User_Refresh_TTL = conv.Duration(7 * 24 * time.Hour) // 7天 = "168h0m0s"
)

// User_type 登录用户完整结构体
type User_type struct {
	Id                uint
	Access_Key        string        // 登录用户名（唯一）
	Login_Pwd_Hash    string        // bcrypt 登录密码哈希
	Enable            int           // 1=启用 0=禁用
	Expire_At         time.Time     // 到期时间（零值=永不过期）
	Access_Token_TTL  conv.Duration // access 令牌时效
	Refresh_Token_TTL conv.Duration // refresh 令牌时效
	Create_Time       time.Time
	Update_Time       time.Time
}

// User_Add_type 登录用户新增/更新结构体（不含 Id/时间）
type User_Add_type struct {
	Access_Key        string
	Login_Pwd_Hash    string
	Enable            int
	Expire_At         time.Time     // 零值=永不过期，落库 NULL
	Access_Token_TTL  conv.Duration // 0 回退默认
	Refresh_Token_TTL conv.Duration // 0 回退默认
}

// User_List_type 登录用户列表视图（脱敏：不含密码哈希）
type User_List_type struct {
	Id                uint
	Access_Key        string
	Enable            int
	Expire_At         time.Time
	Access_Token_TTL  conv.Duration
	Refresh_Token_TTL conv.Duration
	Create_Time       time.Time
	Update_Time       time.Time
}

// User__Query_By_Access_Key 按用户名查询单条登录用户
func User__Query_By_Access_Key(accessKey string) (u User_type, found bool, err error) {
	if accessKey == "" {
		return u, false, fmt.Errorf("User__Query_By_Access_Key: access_key 不能为空")
	}
	query := `
		SELECT Id, Access_Key, Login_Pwd_Hash, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time
		FROM user_account
		WHERE Access_Key = ?
		LIMIT 1
	`
	var (
		pwdHash sql.NullString
		expire  sql.NullTime
		atStr   string
		rtStr   string
	)
	err = DB.QueryRow(query, accessKey).Scan(
		&u.Id, &u.Access_Key, &pwdHash, &u.Enable, &expire,
		&atStr, &rtStr, &u.Create_Time, &u.Update_Time,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return u, false, nil
	}
	if err != nil {
		return u, false, fmt.Errorf("[User__Query_By_Access_Key] 查询失败 | access_key=%s | err=%w", accessKey, err)
	}
	u.Login_Pwd_Hash = pwdHash.String
	u.Access_Token_TTL = parseDur(atStr, Default_User_Access_TTL)
	u.Refresh_Token_TTL = parseDur(rtStr, Default_User_Refresh_TTL)
	if expire.Valid {
		u.Expire_At = expire.Time
	}
	return u, true, nil
}

// User__Add 新增登录用户（Create_Time/Update_Time 取当前时间）
func User__Add(cfg User_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("User__Add: Access_Key 不能为空")
	}
	if cfg.Login_Pwd_Hash == "" {
		return fmt.Errorf("User__Add: Login_Pwd_Hash 不能为空")
	}
	now := time.Now()
	query := `
		INSERT INTO user_account
			(Access_Key, Login_Pwd_Hash, Enable, Expire_At,
			 Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query,
		cfg.Access_Key,
		cfg.Login_Pwd_Hash,
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeDur(cfg.Access_Token_TTL, Default_User_Access_TTL),
		normalizeDur(cfg.Refresh_Token_TTL, Default_User_Refresh_TTL),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("User__Add 新增登录用户失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// User__Update_Mutable 按用户名更新可变字段（密码/启用/到期/时效），刷新 Update_Time
func User__Update_Mutable(cfg User_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("User__Update_Mutable: Access_Key 不能为空")
	}
	query := `
		UPDATE user_account SET
			Login_Pwd_Hash = ?, Enable = ?, Expire_At = ?,
			Access_Token_TTL = ?, Refresh_Token_TTL = ?, Update_Time = ?
		WHERE Access_Key = ?
	`
	_, err = DB.Exec(query,
		cfg.Login_Pwd_Hash,
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeDur(cfg.Access_Token_TTL, Default_User_Access_TTL),
		normalizeDur(cfg.Refresh_Token_TTL, Default_User_Refresh_TTL),
		time.Now(),
		cfg.Access_Key,
	)
	if err != nil {
		return fmt.Errorf("User__Update_Mutable 更新登录用户失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// User__Query 分页查询登录用户列表（脱敏：不返回密码哈希）
func User__Query(page uint, pageSize uint) (list []User_List_type, err error) {
	baseQuery := `
		SELECT Id, Access_Key, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time
		FROM user_account
		ORDER BY Id ASC
	`
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		return list, fmt.Errorf("ERROR 查询登录用户列表失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			user   User_List_type
			expire sql.NullTime
			atStr  string
			rtStr  string
		)
		err = rows.Scan(
			&user.Id, &user.Access_Key, &user.Enable, &expire,
			&atStr, &rtStr, &user.Create_Time, &user.Update_Time,
		)
		if err != nil {
			return list, fmt.Errorf("ERROR 遍历登录用户列表失败，错误:%w", err)
		}
		user.Access_Token_TTL = parseDur(atStr, Default_User_Access_TTL)
		user.Refresh_Token_TTL = parseDur(rtStr, Default_User_Refresh_TTL)
		if expire.Valid {
			user.Expire_At = expire.Time
		}
		list = append(list, user)
	}
	if err = rows.Err(); err != nil {
		return list, fmt.Errorf("ERROR 登录用户列表结果集异常，错误:%w", err)
	}
	return list, nil
}

// User__Count 统计登录用户总数
func User__Count(page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `user_account`"
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[User__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return count, err
}

// User__Del 按 Id 批量删除登录用户
func User__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	for idx, id := range ids {
		if id == 0 {
			return fmt.Errorf("ERROR 第%d条登录用户ID(Id)不能为空", idx+1)
		}
		if _, execErr := DB.Exec("DELETE FROM `user_account` WHERE `Id` = ?", id); execErr != nil {
			return fmt.Errorf("ERROR 第%d条登录用户删除失败, ID:%d, 错误:%w", idx+1, id, execErr)
		}
	}
	return nil
}

// ============================================================
// api_key —— AK/SK 接口账号（供服务间 HMAC 签名通路使用）
// ============================================================

// api_key 令牌时效默认值（以 time.Duration 存储为 duration 字符串，如 "15m0s"）
const (
	Default_ApiKey_Access_TTL  = conv.Duration(15 * time.Minute)   // 900s  = "15m0s"
	Default_ApiKey_Refresh_TTL = conv.Duration(7 * 24 * time.Hour) // 7天 = "168h0m0s"
)

// normalizeDur 将 duration 时效转为入库字符串；0 回退默认值
func normalizeDur(ttl, def conv.Duration) string {
	if ttl == 0 {
		ttl = def
	}
	return time.Duration(ttl).String()
}

// parseDur 解析入库的 duration 字符串；空值/非法回退默认值
func parseDur(s string, def conv.Duration) conv.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return conv.Duration(d)
}

// ApiKey_type AK/SK 账号完整结构体
type ApiKey_type struct {
	Id                uint
	Access_Key        string        // AccessKey（唯一）
	Name              string        // 名称/备注（便于识别与搜索，允许为空）
	Secret_Key        string        // HMAC 签名密钥
	Enable            int           // 1=启用 0=禁用
	Expire_At         time.Time     // 到期时间（零值=永不过期）
	Access_Token_TTL  conv.Duration // access 令牌时效
	Refresh_Token_TTL conv.Duration // refresh 令牌时效
	Create_Time       time.Time
	Update_Time       time.Time
}

// ApiKey_Add_type AK/SK 账号新增/更新/同步结构体（不含 Id/时间）
type ApiKey_Add_type struct {
	Access_Key        string
	Name              string
	Secret_Key        string
	Enable            int
	Expire_At         time.Time
	Access_Token_TTL  conv.Duration // 0 回退默认
	Refresh_Token_TTL conv.Duration // 0 回退默认
}

// ApiKey_List_type AK/SK 账号列表视图（脱敏：不返回 secret_key）
type ApiKey_List_type struct {
	Id                uint
	Access_Key        string
	Name              string
	Enable            int
	Expire_At         time.Time
	Access_Token_TTL  conv.Duration
	Refresh_Token_TTL conv.Duration
	Create_Time       time.Time
	Update_Time       time.Time
}

// ApiKey__Query_By_Access_Key 按 Access_Key 查询单条 AK/SK（HMAC 通路取 Secret_Key 用）
func ApiKey__Query_By_Access_Key(accessKey string) (k ApiKey_type, found bool, err error) {
	if accessKey == "" {
		return k, false, fmt.Errorf("ApiKey__Query_By_Access_Key: access_key 不能为空")
	}
	query := `
		SELECT Id, Access_Key, Name, Secret_Key, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time
		FROM api_key
		WHERE Access_Key = ?
		LIMIT 1
	`
	var (
		expire  sql.NullTime
		nameCol sql.NullString
		atStr   string
		rtStr   string
	)
	err = DB.QueryRow(query, accessKey).Scan(
		&k.Id, &k.Access_Key, &nameCol, &k.Secret_Key, &k.Enable, &expire,
		&atStr, &rtStr, &k.Create_Time, &k.Update_Time,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return k, false, nil
	}
	if err != nil {
		return k, false, fmt.Errorf("[ApiKey__Query_By_Access_Key] 查询失败 | access_key=%s | err=%w", accessKey, err)
	}
	k.Name = nameCol.String
	k.Access_Token_TTL = parseDur(atStr, Default_ApiKey_Access_TTL)
	k.Refresh_Token_TTL = parseDur(rtStr, Default_ApiKey_Refresh_TTL)
	if expire.Valid {
		k.Expire_At = expire.Time
	}
	return k, true, nil
}

// ApiKey__Add 新增 AK/SK 账号
func ApiKey__Add(cfg ApiKey_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("ApiKey__Add: Access_Key 不能为空")
	}
	if cfg.Secret_Key == "" {
		return fmt.Errorf("ApiKey__Add: Secret_Key 不能为空")
	}
	now := time.Now()
	query := `
		INSERT INTO api_key
			(Access_Key, Name, Secret_Key, Enable, Expire_At,
			 Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query,
		cfg.Access_Key,
		cfg.Name,
		cfg.Secret_Key,
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeDur(cfg.Access_Token_TTL, Default_ApiKey_Access_TTL),
		normalizeDur(cfg.Refresh_Token_TTL, Default_ApiKey_Refresh_TTL),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("ApiKey__Add 新增 AK/SK 账号失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// ApiKey__Update_Mutable 按 Access_Key 更新可变字段（Secret_Key/Enable/Expire_At），刷新 Update_Time
func ApiKey__Update_Mutable(cfg ApiKey_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("ApiKey__Update_Mutable: Access_Key 不能为空")
	}
	query := `
		UPDATE api_key SET
			Name = ?, Secret_Key = ?, Enable = ?, Expire_At = ?,
			Access_Token_TTL = ?, Refresh_Token_TTL = ?, Update_Time = ?
		WHERE Access_Key = ?
	`
	_, err = DB.Exec(query,
		cfg.Name,
		cfg.Secret_Key,
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeDur(cfg.Access_Token_TTL, Default_ApiKey_Access_TTL),
		normalizeDur(cfg.Refresh_Token_TTL, Default_ApiKey_Refresh_TTL),
		time.Now(),
		cfg.Access_Key,
	)
	if err != nil {
		return fmt.Errorf("ApiKey__Update_Mutable 更新 AK/SK 账号失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// ApiKey__Sync 按 Access_Key upsert AK/SK 账号（供 internal/sync-aksk 使用）
func ApiKey__Sync(cfgs []ApiKey_Add_type) (added int, updated int, err error) {
	for i, cfg := range cfgs {
		if cfg.Access_Key == "" {
			return added, updated, fmt.Errorf("同步 AK/SK 失败：第%d条 Access_Key 不能为空", i+1)
		}
		if cfg.Secret_Key == "" {
			return added, updated, fmt.Errorf("同步 AK/SK 失败：第%d条 Secret_Key 不能为空", i+1)
		}
		_, found, qErr := ApiKey__Query_By_Access_Key(cfg.Access_Key)
		if qErr != nil {
			return added, updated, qErr
		}
		if found {
			if uErr := ApiKey__Update_Mutable(cfg); uErr != nil {
				return added, updated, uErr
			}
			updated++
		} else {
			if aErr := ApiKey__Add(cfg); aErr != nil {
				return added, updated, aErr
			}
			added++
		}
	}
	log.Printf("同步 AK/SK：新增 %d 条，更新 %d 条", added, updated)
	return added, updated, nil
}

// ApiKey__Query 分页查询 AK/SK 账号列表（脱敏：不返回 secret_key）
func ApiKey__Query(page uint, pageSize uint) (list []ApiKey_List_type, err error) {
	baseQuery := `
		SELECT Id, Access_Key, Name, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time
		FROM api_key
		ORDER BY Id ASC
	`
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		return list, fmt.Errorf("ERROR 查询 AK/SK 列表失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			key     ApiKey_List_type
			expire  sql.NullTime
			nameCol sql.NullString
			atStr   string
			rtStr   string
		)
		err = rows.Scan(
			&key.Id, &key.Access_Key, &nameCol, &key.Enable, &expire,
			&atStr, &rtStr, &key.Create_Time, &key.Update_Time,
		)
		if err != nil {
			return list, fmt.Errorf("ERROR 遍历 AK/SK 列表失败，错误:%w", err)
		}
		key.Name = nameCol.String
		key.Access_Token_TTL = parseDur(atStr, Default_ApiKey_Access_TTL)
		key.Refresh_Token_TTL = parseDur(rtStr, Default_ApiKey_Refresh_TTL)
		if expire.Valid {
			key.Expire_At = expire.Time
		}
		list = append(list, key)
	}
	if err = rows.Err(); err != nil {
		return list, fmt.Errorf("ERROR AK/SK 列表结果集异常，错误:%w", err)
	}
	return list, nil
}

// ApiKey__Count 统计 AK/SK 账号总数
func ApiKey__Count(page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `api_key`"
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[ApiKey__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return count, err
}

// ApiKey__AccessKeysByIds 按 Id 批量取 Access_Key（删除 AK/SK 前用于清理其 Redis 验签缓存）
func ApiKey__AccessKeysByIds(ids []uint) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	query := fmt.Sprintf("SELECT Access_Key FROM api_key WHERE Id IN (%s)", placeholders)
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("[ApiKey__AccessKeysByIds] 查询失败 | err=%w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("[ApiKey__AccessKeysByIds] 遍历失败 | err=%w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ApiKey__Del 按 Id 批量删除 AK/SK 账号
func ApiKey__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	for idx, id := range ids {
		if id == 0 {
			return fmt.Errorf("ERROR 第%d条 AK/SK ID(Id)不能为空", idx+1)
		}
		if _, execErr := DB.Exec("DELETE FROM `api_key` WHERE `Id` = ?", id); execErr != nil {
			return fmt.Errorf("ERROR 第%d条 AK/SK 删除失败, ID:%d, 错误:%w", idx+1, id, execErr)
		}
	}
	return nil
}
