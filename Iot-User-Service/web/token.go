/*
* 日期: 2026.3.13 PM 2:20
* 作者: 范范zwf
* 作用: web层的token相关代码
 */
package web

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"main/cloud"
	"strings"
	"time"
)

// ====================== 生成RSA密钥 ======================
// 参数：bits（密钥长度，推荐2048或以上）
// 返回：生成的RSA私钥和公钥
func Generate_RSA_Key_Pair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("生成私钥失败: %v", err)
	}
	// 从私钥中提取公钥
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey, nil
}

// 将rsa.PrivateKey序列化为PEM格式字符串（方便存储）
func Private_Key_ToPEM(privateKey *rsa.PrivateKey) (string, error) {
	// 把私钥转为ASN.1 DER编码
	derBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if derBytes == nil {
		return "", errors.New("私钥DER编码失败")
	}
	// 封装为PEM格式
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	}
	// 转为字符串
	return string(pem.EncodeToMemory(pemBlock)), nil
}

// 将rsa.PublicKey序列化为PEM格式字符串（方便存储）
func Public_Key_ToPEM(publicKey *rsa.PublicKey) (string, error) {
	// 把公钥转为ASN.1 DER编码
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("公钥DER编码失败: %v", err)
	}
	// 封装为PEM格式
	pemBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: derBytes,
	}
	// 转为字符串
	return string(pem.EncodeToMemory(pemBlock)), nil
}

// ========== PEM字符串转回密钥结构体 ==========
// PEM字符串转rsa.PrivateKey
func PEMTo_Private_Key(pemStr string) (*rsa.PrivateKey, error) {
	// 1. 解析PEM块
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("解析PEM私钥失败")
	}
	// 2. DER编码转rsa.PrivateKey
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("私钥解析失败: %v", err)
	}
	return privateKey, nil
}

// PEM字符串转rsa.PublicKey
func PEMTo_Public_Key(pemStr string) (*rsa.PublicKey, error) {
	// 1. 解析PEM块
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("解析PEM公钥失败")
	}
	// 2. DER编码转rsa.PublicKey
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("公钥解析失败: %v", err)
	}
	// 3. 类型断言转为rsa.PublicKey
	publicKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("公钥类型不是RSA")
	}
	return publicKey, nil
}

// ====================== 随机盐生成工具 ======================
// generateRandomSalt 生成指定长度的随机盐（默认16字节）
func generate_Random_Salt(length int) ([]byte, error) {
	if length <= 0 {
		length = 16 // 默认16字节随机盐
	}
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("ERROR 生成随机盐失败: %v", err)
	}
	return salt, nil
}

// ====================== Token 核心函数（关键修改） ======================
// CreateShortToken 创建带随机盐的Token
// 参数：
// 	salt_length: 随机盐长度（可选，默认16字节）
// 	private_key: RSA私钥
// 	encrypted_user_info: AES加密后的用户信息
// 	issue_time: 签发时间
// 	expire_time: 过期时间

// 返回：Token字符串 | 错误
func Create_Short_Token(salt_length int, private_key *rsa.PrivateKey, encrypted_user_info []byte, issue_time time.Time, expire_time time.Time) (string, error) {
	// 1. 生成16字节随机盐
	salt, err := generate_Random_Salt(salt_length)
	if err != nil {
		return "", err
	}
	// 随机盐转base64（便于拼接和传输）
	base64Salt := base64.URLEncoding.EncodeToString(salt)

	// 2. 时间转秒级时间戳
	issueTs := issue_time.Format(time.RFC3339Nano)   // 可读时间格式
	expireTs := expire_time.Format(time.RFC3339Nano) // 可读时间格式

	// 3. 加密用户信息转base64
	base64UserInfo := base64.URLEncoding.EncodeToString(encrypted_user_info)

	// 4. 拼接待签名内容（包含随机盐，分隔符用.）
	// 内容：用户信息 + 签发时间 + 过期时间 + 随机盐
	signContent := fmt.Sprintf("%s.%s.%s.%s", base64UserInfo, issueTs, expireTs, base64Salt)

	// 5. SHA256哈希 + RSA私钥签名
	hash := sha256.New()
	hash.Write([]byte(signContent))
	signature, err := rsa.SignPKCS1v15(rand.Reader, private_key, crypto.SHA256, hash.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("ERROR 签名失败: %v", err)
	}
	base64Signature := base64.URLEncoding.EncodeToString(signature)

	// 6. 最终Token结构（用.分隔）：
	// base64(加密用户信息).签发时间戳.过期时间戳.base64(随机盐).base64(签名)
	token := fmt.Sprintf("%s.%s.%s.%s.%s", base64UserInfo, Base64Encode(issueTs), Base64Encode(expireTs), base64Salt, base64Signature)
	return Base64Encode(token), nil
}

// VerifyShortToken 验证带随机盐的Token
// 参数：
//
//	public_Key: RSA公钥
//	token: 待验证的Token字符串
//
// 返回：AES加密后的用户信息 | 错误
func Verify_Short_Token(public_key *rsa.PublicKey, token string) ([]byte, error) {
	token_decode, err := Base64Decode(token)
	if err != nil {
		return nil, fmt.Errorf("ERROR 签发时间解码失败: %v", err)
	}

	// 1. 拆分Token（按.分隔，需拆分为5部分）
	parts := strings.Split(token_decode, ".")
	if len(parts) != 5 {
		return nil, fmt.Errorf("ERROR Token格式错误，需为5部分 %d", len(parts))
	}
	// 提取各部分
	base64UserInfo := parts[0]
	base64Salt := parts[3]
	base64Signature := parts[4]

	issueTsStr, err := Base64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("ERROR 签发时间解码失败: %v", err)
	}

	expireTsStr, err := Base64Decode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("ERROR 签发时间解码失败: %v", err)
	}

	// 2. 验证时间有效性
	issueTs, err := time.Parse(time.RFC3339Nano, issueTsStr)
	if err != nil {
		return nil, fmt.Errorf("ERROR 签发时间解析失败: %v", err)
	}

	expireTs, err := time.Parse(time.RFC3339Nano, expireTsStr)
	if err != nil {
		return nil, fmt.Errorf("ERROR 过期时间解析失败: %v", err)
	}

	now := time.Now()

	if now.After(expireTs) { // t1 晚于 t2（t1 > t2）
		return nil, errors.New("WARNING Token已过期")
	} else if now.Before(issueTs) { // t1 早于 t2（t1 < t2）
		return nil, errors.New("ERROR Token签发时间异常")
	}

	// 3. 还原待签名内容（和创建时一致，包含随机盐）
	signContent := fmt.Sprintf("%s.%s.%s.%s", base64UserInfo, issueTsStr, expireTsStr, base64Salt)

	// 4. 解码签名
	signature, err := base64.URLEncoding.DecodeString(base64Signature)
	if err != nil {
		return nil, fmt.Errorf("ERROR 签名解码失败: %v", err)
	}

	// 5. RSA验签
	hash := sha256.New()
	hash.Write([]byte(signContent))
	err = rsa.VerifyPKCS1v15(public_key, crypto.SHA256, hash.Sum(nil), signature)
	if err != nil {
		return nil, fmt.Errorf("ERROR Token签名验证失败: %v", err)
	}

	// 6. 解码并返回加密用户信息
	encryptedUserInfo, err := base64.URLEncoding.DecodeString(base64UserInfo)
	if err != nil {
		return nil, fmt.Errorf("ERROR 用户信息解码失败: %v", err)
	}
	return encryptedUserInfo, nil
}

// ====================== AES 辅助函数（无修改） ======================
func Aes_Encrypt(plainText []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plainText = pkcs7Padding(plainText, block.BlockSize())
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	cipherText := make([]byte, len(plainText))
	mode.CryptBlocks(cipherText, plainText)
	return append(iv, cipherText...), nil
}

func Aes_Decrypt(cipherText []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := cipherText[:block.BlockSize()]
	cipherText = cipherText[block.BlockSize():]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)
	plainText = pkcs7Unpadding(plainText)
	return plainText, nil
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7Unpadding(data []byte) []byte {
	length := len(data)
	padding := int(data[length-1])
	return data[:length-padding]
}

// Base64Encode 标准Base64编码（输出不含空格/点，包含+、/、=）
func Base64Encode(input string) string {
	// 将字符串转字节数组后编码
	encoded := base64.StdEncoding.EncodeToString([]byte(input))
	return encoded
}

// Base64Decode 标准Base64解码
func Base64Decode(encodedStr string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败：%v", err)
	}
	return string(decodedBytes), nil
}

// ======================  api中的token ======================

// api中的token中的api信息结构体
type Token_Api_Info struct {
	Api_Id   uint   // 接口Id
	Login_Ip string // 登录ip

}

// 信息结构体转json并AES加密
func Token_Api_Info__Json_AES_Encrypt(Aes string, info Token_Api_Info) (aes_data []byte, err error) {
	jsonByte, err := json.Marshal(info)
	if err != nil {
		err = fmt.Errorf("ERROR Token_Api_Info结构体转json失败: %v", err)
		return
	}

	aes_data, err = cloud.AesEncryptGCM(jsonByte, Aes) // AES加密（GCM模式 - 推荐）
	return
}

// 信息数据AES解密并转结构体
func Token_Api__Info__Json_AES_Decrypt(Aes string, d []byte) (info Token_Api_Info, err error) {
	cloud_Aes_Decrypt, err := cloud.AesDecryptGCM(d, Aes) // AES解密（GCM模式 - 推荐）
	if err != nil {
		err = fmt.Errorf("ERROR AES解密失败: %v", err)
		return
	}

	err = json.Unmarshal(cloud_Aes_Decrypt, &info)
	if err != nil {
		err = fmt.Errorf("ERROR JSON解析失败: %v", err)
	}
	return
}

// ======================  用户中的token ======================

// api中的token中的api信息结构体
type Token_User_Info struct {
	User_Id  uint   // 接口Id
	Login_Ip string // 登录ip

}

// 信息结构体转json并AES加密
func Token_User_Info__Json_AES_Encrypt(Aes string, info Token_User_Info) (aes_data []byte, err error) {
	jsonByte, err := json.Marshal(info)
	if err != nil {
		err = fmt.Errorf("ERROR Token_User_Info结构体转json失败: %v", err)
		return
	}

	aes_data, err = cloud.AesEncryptGCM(jsonByte, Aes) // AES加密（GCM模式 - 推荐）
	return
}

// 信息数据AES解密并转结构体
func Token_User_Info__Json_AES_Decrypt(Aes string, d []byte) (info Token_User_Info, err error) {
	cloud_Aes_Decrypt, err := cloud.AesDecryptGCM(d, Aes) // AES解密（GCM模式 - 推荐）
	if err != nil {
		err = fmt.Errorf("ERROR AES解密失败: %v", err)
		return
	}

	err = json.Unmarshal(cloud_Aes_Decrypt, &info)
	if err != nil {
		err = fmt.Errorf("ERROR JSON解析失败: %v", err)
	}
	return
}
