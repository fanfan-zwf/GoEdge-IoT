/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: redis 用户逻辑
 */

package redis

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var (
	Default_Expiration_Second time.Duration = 60 * 60 * time.Second // 默认时间
	Short_Expiration_Second   time.Duration = 60 * 60 * time.Second // 短时间
)

/*
***************刷新令牌***************
 */
// 访问令牌结构体
type Refresh_Token_redis_type struct {
	User_Id       uint      // 用户id
	Terminal_Uuid string    // 用户终端Id
	Expires_in    time.Time // 访问令牌过期时间
	Login_Ip      string    // 登录ip

	Salt               string // 随机盐
	RSA_PrivateKeyPath string // RSA私钥路径
	RSA_PublicKeyPath  string // RSA公钥路径
}

// 创建访问令牌
func Refresh_Token_Add(User_Id uint, Refresh_Token string, Value Refresh_Token_redis_type, Expiration time.Duration) (err error) {
	if User_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Refresh_Token:%d:%s", User_Id, Refresh_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), Expiration).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Refresh_Token_Query(User_Id uint, Refresh_Token string) (Access_Token_redis Refresh_Token_redis_type, err error) {
	if User_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	key := fmt.Sprintf("Refresh_Token:%d:%s", User_Id, Refresh_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************访问令牌***************
 */
// 访问令牌结构体
type Access_Token_redis_type struct {
	User_Id       uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
	Login_Ip      string    // 登录ip

	Salt               string // 随机盐
	RSA_PrivateKeyPath string // RSA私钥路径
	RSA_PublicKeyPath  string // RSA公钥路径
}

// 创建访问令牌
func Access_Token_Add(Access_Token string, Value Access_Token_redis_type) (err error) {
	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Access_Token:%s", Access_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), time.Until(Value.Expires_in)).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Access_Token_Query(Access_Token string) (Access_Token_redis Access_Token_redis_type, err error) {

	key := fmt.Sprintf("Access_Token:%s", Access_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************用户权限***************
 */
// 读取用户权限
func Authority_User_Query(User_Id uint, Authority_Theme string) (exist bool, err error) {
	var value string
	User_Id_str := fmt.Sprintf("%d", User_Id)
	value, err = Rdb.HGet(
		ctx,
		User_Id_str, // 用户id
		fmt.Sprintf("Authority_User:%s", Authority_Theme), // 权限主题
	).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = Rdb.Expire(ctx, User_Id_str, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	switch value {
	case "true":
		exist = true
	case "false":
		exist = false
	default:
		err = fmt.Errorf("未知状态 %v", value)
	}

	return
}

// 写入用户权限
func Authority_User_Add(User_Id uint, Authority_Theme string, exist bool) (err error) {
	User_Id_str := fmt.Sprintf("%d", User_Id)
	err = Rdb.HSet(
		ctx,
		User_Id_str, // 用户id
		fmt.Sprintf("Authority_User:%s", Authority_Theme), // 权限主题
		fmt.Sprintf("%t", exist),
	).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = Rdb.Expire(ctx, User_Id_str, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	return
}

// 删除指定的value
func Authority_User_Del_Value(User_Id uint, Authority_Theme ...string) (err error) {
	for i, v := range Authority_Theme {
		Authority_Theme[i] = fmt.Sprintf("Authority_User:%s", v)
	}

	User_Id_str := fmt.Sprintf("%d", User_Id)
	_, err = Rdb.HDel(
		ctx,
		User_Id_str,        // 用户id
		Authority_Theme..., // 权限主题
	).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************用户密码输入错误锁定***************
 */

// 用户输入错误记录
func User_Passwd_Input_Error_Record(User_Id uint, Value string, Limit_TTL_Second uint) (length int, err error) {
	key := fmt.Sprintf("User_Passwd_Input_Error:%d", User_Id)

	var val int64
	val, err = Rdb.LPush(ctx, key, Value).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	length = int(val)

	err = Rdb.Expire(ctx, key, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	return
}

func User_Passwd_Input_Error_Record_Del(User_Id uint) (err error) {
	key := fmt.Sprintf("User_Passwd_Input_Error:%d", User_Id)
	err = Rdb.Del(ctx, key).Err()
	return
}

/*
***************刷新令牌***************
 */
// 访问令牌结构体
type Api_Refresh_Token_redis_type struct {
	Api_Id     uint      // 用户id
	Expires_in time.Time // 访问令牌过期时间
	Allow_Ip   string    // 允许ip
	Login_Ip   string    // 登录ip

	Salt            string // 随机盐
	RSA_Private_Key string // RSA私钥
	RSA_Public_Key  string // RSA公钥
}

// 创建访问令牌
func Api_Refresh_Token_Add(Refresh_Token string, Value Api_Refresh_Token_redis_type) (err error) {
	if Value.Api_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Api_Refresh_Token:%d:%s", Value.Api_Id, Refresh_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), time.Until(Value.Expires_in)).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Api_Refresh_Token_Query(Api_Id uint, Refresh_Token string) (Refresh_Token_redis Api_Refresh_Token_redis_type, err error) {
	if Api_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	key := fmt.Sprintf("Api_Refresh_Token:%d:%s", Api_Id, Refresh_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Refresh_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************访问令牌***************
 */
// 访问令牌结构体
type Api_Access_Token_redis_type struct {
	Api_Id        uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
	Allow_Ip      string    // 允许ip
	Login_Ip      string    // 登录ip

	Salt            string // 随机盐
	RSA_Private_Key string // RSA私钥
	RSA_Public_Key  string // RSA公钥
}

// 创建访问令牌
func Api_Access_Token_Add(Access_Token string, Value Api_Access_Token_redis_type) (err error) {
	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Api_Access_Token:%s", Access_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), time.Until(Value.Expires_in)).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Api_Access_Token_Query(Access_Token string) (Access_Token_redis Api_Access_Token_redis_type, err error) {

	key := fmt.Sprintf("Api_Access_Token:%s", Access_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}
