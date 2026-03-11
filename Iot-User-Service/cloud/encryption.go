/*
* 日期: 2025.10.10 PM 10:58
* 作者: 范范zwf
* 作用: 加密
 */

package cloud

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

// Gzip压缩
// 传入原始数据，输出加密数据和错误
func Gzip_compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Gzip解压
// 传入加密数据和限制大小，输出原始数据和错误
// 限制大小为字节，如: 10M 10*1024*1024
func Gzip_decompress(compressed []byte, maxSize int64) ([]byte, error) {
	// 限制解压大小防止炸弹攻击
	limitReader := io.LimitReader(bytes.NewReader(compressed), maxSize) // 10MB

	gz, err := gzip.NewReader(limitReader)
	if err != nil {
		return []byte{}, err
	}
	defer gz.Close()

	var result bytes.Buffer
	_, err = io.Copy(&result, gz)
	if err != nil {
		return []byte{}, err
	}

	return result.Bytes(), nil
}

// AES加密（GCM模式 - 推荐）
func AesEncryptGCM(plainText []byte, key string) ([]byte, error) {
	hash_key := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hash_key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plainText, nil), nil
}

// AES解密（GCM模式）
func AesDecryptGCM(cipherText []byte, key string) ([]byte, error) {
	hash_key := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hash_key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, errors.New("cipherText too short")
	}

	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	return gcm.Open(nil, nonce, cipherText, nil)
}
