/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 密钥生成工具。GenKey 生成密码学随机字节，并以 base64url（无填充）输出，
*       用于 AK、SK、access_token、refresh_token、nonce 等标识/凭证。
 */
package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenKey 生成 nBytes 个密码学随机字节，输出 base64url（URL 安全、无 '=' 填充）字符串。
// 传参 nBytes：随机字节数（<=0 视为非法）；返回 base64url 文本与错误。
// 说明：AK/SK 建议 16 字节，token/nonce 建议 32 字节，长度可自定义。
func GenKey(nBytes int) (string, error) {
	if nBytes <= 0 {
		return "", fmt.Errorf("GenKey: 字节长度必须大于 0，当前=%d", nBytes)
	}

	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("GenKey: 生成随机字节失败: %w", err)
	}

	// RawURLEncoding = URL 安全的 base64（- 和 _ 替代 + 和 /），且不添加 '=' 填充
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
