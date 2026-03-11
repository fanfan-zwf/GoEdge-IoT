/*
* 日期: 2025.10.10 PM 10:58
* 作者: 范范zwf
* 作用: 加密
 */

package cloud

import (
	"bytes"
	"compress/gzip" // ✅ Added missing gzip import (also standard library)
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash/crc32" // ✅ Corrected import path (not crypto/crc32)
	"io"
	"strconv"
)

// Gzip压缩
func GzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	defer gz.Close()

	_, err := gz.Write(data)
	if err != nil {
		return nil, fmt.Errorf("gzip写入失败: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip关闭失败: %w", err)
	}

	return buf.Bytes(), nil
}

// Gzip解压
func GzipDecompress(compressed []byte, maxSize int64) ([]byte, error) {
	limitReader := io.LimitReader(bytes.NewReader(compressed), maxSize)

	gz, err := gzip.NewReader(limitReader)
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gz.Close()

	var result bytes.Buffer
	_, err = io.Copy(&result, gz)
	if err != nil {
		return nil, fmt.Errorf("gzip解压读取失败: %w", err)
	}

	return result.Bytes(), nil
}

// AES加密（GCM模式）
func AesEncryptGCM(plainText []byte, key string) ([]byte, error) {
	if key == "" {
		return plainText, nil
	}
	hashKey := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hashKey[:])
	if err != nil {
		return nil, fmt.Errorf("创建AES块失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	return gcm.Seal(nonce, nonce, plainText, nil), nil
}

// AES解密（GCM模式）
func AesDecryptGCM(cipherText []byte, key string) ([]byte, error) {
	if key == "" {
		return cipherText, nil
	}
	hashKey := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hashKey[:])
	if err != nil {
		return nil, fmt.Errorf("创建AES块失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, errors.New("密文长度不足，无法拆分nonce")
	}

	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM解密失败: %w", err)
	}
	return plainText, nil
}

// 全局CRC32查表（正确使用hash/crc32）
var crcTable = crc32.MakeTable(crc32.IEEE)

// EncodeWithCRC32 追加CRC32校验
func EncodeWithCRC32(jsonBytes []byte) ([]byte, error) {
	if len(jsonBytes) == 0 {
		return nil, fmt.Errorf("输入JSON字节流为空")
	}

	crcValue := crc32.Checksum(jsonBytes, crcTable)
	crcBytes := []byte(strconv.FormatUint(uint64(crcValue), 10))

	result := make([]byte, 0, len(jsonBytes)+1+len(crcBytes))
	result = append(result, jsonBytes...)
	result = append(result, '|')
	result = append(result, crcBytes...)

	return result, nil
}

// DecodeAndVerifyCRC32 校验CRC32
func DecodeAndVerifyCRC32(dataBytes []byte) ([]byte, error) {
	if len(dataBytes) == 0 {
		return nil, fmt.Errorf("输入字节流为空")
	}

	sepIndex := bytes.LastIndexByte(dataBytes, '|')
	if sepIndex == -1 {
		return nil, fmt.Errorf("数据格式错误：未找到分隔符|")
	}
	if sepIndex == len(dataBytes)-1 {
		return nil, fmt.Errorf("数据格式错误：分隔符|后无CRC32值")
	}

	jsonBytes := dataBytes[:sepIndex]
	crcBytes := dataBytes[sepIndex+1:]

	crcStr := string(crcBytes)
	receivedCRC, err := strconv.ParseUint(crcStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("解析CRC32值失败：%w", err)
	}

	calculatedCRC := crc32.Checksum(jsonBytes, crcTable)
	if calculatedCRC != uint32(receivedCRC) {
		return nil, fmt.Errorf("CRC32校验失败：接收值=%d，计算值=%d", receivedCRC, calculatedCRC)
	}

	return jsonBytes, nil
}

// Send__CRC32_Aes_Gzip 发送端处理
func Send__CRC32_Aes_Gzip(dataBytes []byte, aesPasswd string) ([]byte, error) {
	gzipData, err := GzipCompress(dataBytes)
	if err != nil {
		return nil, fmt.Errorf("压缩失败: %w", err)
	}

	aesData, err := AesEncryptGCM(gzipData, aesPasswd)
	if err != nil {
		return nil, fmt.Errorf("加密失败: %w", err)
	}

	crcData, err := EncodeWithCRC32(aesData)
	if err != nil {
		return nil, fmt.Errorf("添加CRC32失败: %w", err)
	}

	return crcData, nil
}

// Receive__CRC32_Aes_Gzip 接收端处理
func Receive__CRC32_Aes_Gzip(dataBytes []byte, aesPasswd string) ([]byte, error) {
	crcData, err := DecodeAndVerifyCRC32(dataBytes)
	if err != nil {
		return nil, fmt.Errorf("CRC32校验失败: %w", err)
	}

	aesData, err := AesDecryptGCM(crcData, aesPasswd)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	gzipData, err := GzipDecompress(aesData, 20*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("解压失败: %w", err)
	}

	return gzipData, nil
}
