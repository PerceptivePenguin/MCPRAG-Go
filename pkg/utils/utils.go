// Package utils 提供了项目中使用的通用工具函数
//
// 这个包包含了各种实用的工具函数，包括：
// - 字符串处理工具
// - 时间处理工具  
// - 数据转换工具
// - 其他通用工具函数
//
// 通过将这些工具函数集中在一个包中，我们可以：
// - 避免代码重复
// - 提供一致的API接口
// - 集中维护和优化
// - 便于单元测试
package utils

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Ordered 定义了支持比较操作的类型约束
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
	~float32 | ~float64 |
	~string
}

// GeneralUtils 通用工具函数

// Ptr 返回值的指针
func Ptr[T any](v T) *T {
	return &v
}

// Deref 解引用指针，如果为nil则返回零值
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// DerefOr 解引用指针，如果为nil则返回默认值
func DerefOr[T any](p *T, defaultValue T) T {
	if p == nil {
		return defaultValue
	}
	return *p
}

// SafeDeref 安全解引用，返回值和是否为nil的标志
func SafeDeref[T any](p *T) (T, bool) {
	if p == nil {
		var zero T
		return zero, false
	}
	return *p, true
}

// Min 返回两个值中的最小值
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max 返回两个值中的最大值
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp 将值限制在指定范围内
func Clamp[T Ordered](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Abs 返回数值的绝对值
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// AbsFloat 返回浮点数的绝对值
func AbsFloat(x float64) float64 {
	return math.Abs(x)
}

// Round 四舍五入到指定小数位
func Round(x float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(x*multiplier) / multiplier
}

// MD5Hash 计算字符串的MD5哈希值
func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash 计算字符串的SHA256哈希值
func SHA256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DirExists 检查目录是否存在
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(path string) error {
	if !DirExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileExt 获取文件扩展名
func GetFileExt(path string) string {
	return filepath.Ext(path)
}

// GetFileName 获取文件名（不包含路径）
func GetFileName(path string) string {
	return filepath.Base(path)
}

// GetFileNameWithoutExt 获取文件名（不包含路径和扩展名）
func GetFileNameWithoutExt(path string) string {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)]
}

// JoinPath 连接路径
func JoinPath(paths ...string) string {
	return filepath.Join(paths...)
}

// GetCurrentDir 获取当前目录
func GetCurrentDir() (string, error) {
	return os.Getwd()
}

// GetHomeDir 获取用户主目录
func GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetTempDir 获取临时目录
func GetTempDir() string {
	return os.TempDir()
}

// GetExecutableDir 获取可执行文件所在目录
func GetExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

// InSlice 检查元素是否在切片中
func InSlice[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// UniqueSlice 去除切片中的重复元素
func UniqueSlice[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0)
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// ReverseSlice 反转切片
func ReverseSlice[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, j := 0, len(slice)-1; i < len(slice); i, j = i+1, j-1 {
		result[i] = slice[j]
	}
	return result
}

// ChunkSlice 将切片分割成指定大小的块
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		return nil
	}
	
	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	
	return chunks
}

// SafeGo 安全的goroutine启动，捕获panic
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in goroutine: %v\n", r)
			}
		}()
		fn()
	}()
}

// SafeGoWithContext 带上下文的安全goroutine启动
func SafeGoWithContext(ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in goroutine: %v\n", r)
			}
		}()
		fn(ctx)
	}()
}

// GetGoroutineID 获取当前goroutine ID（仅用于调试）
func GetGoroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// 解析 "goroutine ID [running]:" 格式
	stack := string(buf[:n])
	var id int
	fmt.Sscanf(stack, "goroutine %d ", &id)
	return id
}

// WaitGroup 包装sync.WaitGroup提供更方便的API
type WaitGroup struct {
	wg sync.WaitGroup
}

// NewWaitGroup 创建新的WaitGroup
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

// Go 启动goroutine并自动Add(1)
func (w *WaitGroup) Go(fn func()) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		fn()
	}()
}

// GoWithContext 启动带上下文的goroutine
func (w *WaitGroup) GoWithContext(ctx context.Context, fn func(context.Context)) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		fn(ctx)
	}()
}

// SafeGo 启动安全的goroutine（捕获panic）
func (w *WaitGroup) SafeGo(fn func()) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in WaitGroup goroutine: %v\n", r)
			}
		}()
		fn()
	}()
}

// Wait 等待所有goroutine完成
func (w *WaitGroup) Wait() {
	w.wg.Wait()
}

// WaitWithContext 带超时的等待
func (w *WaitGroup) WaitWithContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Once 包装sync.Once提供更方便的API
type Once struct {
	once sync.Once
}

// NewOnce 创建新的Once
func NewOnce() *Once {
	return &Once{}
}

// Do 执行函数，确保只执行一次
func (o *Once) Do(fn func()) {
	o.once.Do(fn)
}

// DoWithError 执行可能返回错误的函数，确保只执行一次
func (o *Once) DoWithError(fn func() error) error {
	var err error
	o.once.Do(func() {
		err = fn()
	})
	return err
}

// Retry 重试执行函数
func Retry(attempts int, fn func() error) error {
	var lastErr error
	
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	
	return lastErr
}

// RetryWithDelay 带延迟的重试执行
func RetryWithDelay(attempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < attempts-1 {
				Sleep(delay)
			}
			continue
		}
		return nil
	}
	
	return lastErr
}