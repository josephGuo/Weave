package pkg

import (
	cryptoRand "crypto/rand"
	"math/big"
	"time"
)

// 随机字符串的字符集（大小写字母 + 数字）
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString 生成指定长度的随机字符串
func RandomString(n int) string {
	if n <= 0 {
		return ""
	}

	b := make([]byte, n)
	cryptoRand.Read(b)

	// 随机字节映射
	for i := range b {
		b[i] = letterBytes[int(b[i])%len(letterBytes)]
	}
	return string(b)
}

// SecureRandomString 使用密码学安全的随机数生成器生成随机字符串
// 保留此函数以保持兼容性
func SecureRandomString(n int) string {
	return RandomString(n)
}

// StrSliceContains 检查字符串切片是否包含指定字符串
func StrSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// StrSliceContainsAny 检查字符串切片是否包含任何一个指定的字符串
func StrSliceContainsAny(slice []string, items ...string) bool {
	for _, item := range items {
		if StrSliceContains(slice, item) {
			return true
		}
	}
	return false
}

// GenerateUniqueID 生成唯一ID（基于时间戳和随机数）
// 格式: YYYYMMDDHHmmss-8位随机字符串
func GenerateUniqueID() string {
	return time.Now().Format("20060102150405") + "-" + RandomString(8)
}

// GenerateShortID 生成更短的唯一ID
// 格式: 时间戳的base36编码-6位随机字符串
func GenerateShortID() string {
	// 使用时间戳的base36编码作为前缀，比标准时间格式更短
	timestamp := time.Now().Unix()
	base36Time := big.NewInt(timestamp).Text(36)
	return base36Time + "-" + RandomString(6)
}

// GenerateRequestID 生成请求ID
// 格式: req-时间戳-12位随机字符串
func GenerateRequestID() string {
	return "req-" + time.Now().Format("20060102150405") + "-" + RandomString(12)
}

// StringInSlice 检查字符串是否在切片中（别名函数，为了兼容）
func StringInSlice(str string, list []string) bool {
	return StrSliceContains(list, str)
}
