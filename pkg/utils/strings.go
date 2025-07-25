package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// StringUtils 字符串工具函数

// IsEmpty 检查字符串是否为空或只包含空格
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty 检查字符串是否非空
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// Truncate 截断字符串到指定长度
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// TruncateWithSuffix 截断字符串并添加后缀
func TruncateWithSuffix(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	
	if len(suffix) >= maxLen {
		return suffix[:maxLen]
	}
	
	truncLen := maxLen - len(suffix)
	if truncLen <= 0 {
		return suffix
	}
	
	return s[:truncLen] + suffix
}

// PadLeft 左填充字符串
func PadLeft(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	
	padding := strings.Repeat(string(padChar), length-len(s))
	return padding + s
}

// PadRight 右填充字符串
func PadRight(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	
	padding := strings.Repeat(string(padChar), length-len(s))
	return s + padding
}

// Center 居中字符串
func Center(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	
	totalPad := length - len(s)
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad
	
	left := strings.Repeat(string(padChar), leftPad)
	right := strings.Repeat(string(padChar), rightPad)
	
	return left + s + right
}

// Reverse 反转字符串
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Contains 检查字符串是否包含子字符串（忽略大小写）
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// StartsWithIgnoreCase 检查字符串是否以指定前缀开始（忽略大小写）
func StartsWithIgnoreCase(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// EndsWithIgnoreCase 检查字符串是否以指定后缀结束（忽略大小写）
func EndsWithIgnoreCase(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}

// EqualsIgnoreCase 比较两个字符串是否相等（忽略大小写）
func EqualsIgnoreCase(s1, s2 string) bool {
	return strings.EqualFold(s1, s2)
}

// SplitAndTrim 分割字符串并去除空格
func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	
	return result
}

// JoinNonEmpty 连接非空字符串
func JoinNonEmpty(sep string, strs ...string) string {
	var nonEmpty []string
	
	for _, s := range strs {
		if IsNotEmpty(s) {
			nonEmpty = append(nonEmpty, s)
		}
	}
	
	return strings.Join(nonEmpty, sep)
}

// CamelToSnake 驼峰转蛇形命名
func CamelToSnake(s string) string {
	var result strings.Builder
	
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	
	return result.String()
}

// SnakeToCamel 蛇形转驼峰命名
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}
	
	var result strings.Builder
	result.WriteString(strings.ToLower(parts[0]))
	
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result.WriteString(strings.ToUpper(parts[i][:1]))
			if len(parts[i]) > 1 {
				result.WriteString(strings.ToLower(parts[i][1:]))
			}
		}
	}
	
	return result.String()
}

// PascalCase 转换为帕斯卡命名（首字母大写的驼峰）
func PascalCase(s string) string {
	camel := SnakeToCamel(s)
	if len(camel) > 0 {
		return strings.ToUpper(camel[:1]) + camel[1:]
	}
	return camel
}

// RemoveNonAlphanumeric 移除非字母数字字符
func RemoveNonAlphanumeric(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return reg.ReplaceAllString(s, "")
}

// RemoveExtraSpaces 移除多余的空格
func RemoveExtraSpaces(s string) string {
	reg := regexp.MustCompile(`\s+`)
	return reg.ReplaceAllString(strings.TrimSpace(s), " ")
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

// MaskString 掩码字符串，保留前后几位，中间用星号替换
func MaskString(s string, keepStart, keepEnd int, maskChar rune) string {
	if len(s) <= keepStart+keepEnd {
		return strings.Repeat(string(maskChar), len(s))
	}
	
	start := s[:keepStart]
	end := s[len(s)-keepEnd:]
	middle := strings.Repeat(string(maskChar), len(s)-keepStart-keepEnd)
	
	return start + middle + end
}

// RepeatString 重复字符串指定次数
func RepeatString(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}

// WordCount 统计单词数量
func WordCount(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}

// LineCount 统计行数
func LineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// FirstNonEmpty 返回第一个非空字符串
func FirstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if IsNotEmpty(s) {
			return s
		}
	}
	return ""
}

// DefaultIfEmpty 如果字符串为空则返回默认值
func DefaultIfEmpty(s, defaultValue string) string {
	if IsEmpty(s) {
		return defaultValue
	}
	return s
}