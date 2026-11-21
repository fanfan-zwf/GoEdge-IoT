/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 网页登录密码的 bcrypt 哈希与校验工具。哈希结果用于 account.Login_Pwd_Hash，
*       明文密码不落库、不下发。
 */
package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt（DefaultCost）对明文密码做哈希，返回可直接入库的哈希串。
func HashPassword(pwd string) (string, error) {
	if pwd == "" {
		return "", fmt.Errorf("HashPassword: 密码不能为空")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("HashPassword: 生成 bcrypt 哈希失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 校验明文密码与已存 bcrypt 哈希是否匹配，匹配返回 true。
func VerifyPassword(hash, pwd string) bool {
	if hash == "" || pwd == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)) == nil
}
