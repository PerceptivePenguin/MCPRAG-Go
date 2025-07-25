package utils

import (
	"fmt"
	"time"
)

// TimeUtils 时间工具函数

// Now 获取当前时间
func Now() time.Time {
	return time.Now()
}

// NowUTC 获取当前UTC时间
func NowUTC() time.Time {
	return time.Now().UTC()
}

// ParseTime 解析时间字符串
func ParseTime(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// FormatTime 格式化时间
func FormatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

// FormatDuration 格式化持续时间为人类可读格式
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := int(d.Hours() / 24)
		hours := d.Hours() - float64(days*24)
		return fmt.Sprintf("%dd %.1fh", days, hours)
	}
}

// IsZeroTime 检查时间是否为零值
func IsZeroTime(t time.Time) bool {
	return t.IsZero()
}

// IsNotZeroTime 检查时间是否非零值
func IsNotZeroTime(t time.Time) bool {
	return !t.IsZero()
}

// IsBefore 检查时间是否在另一个时间之前
func IsBefore(t1, t2 time.Time) bool {
	return t1.Before(t2)
}

// IsAfter 检查时间是否在另一个时间之后
func IsAfter(t1, t2 time.Time) bool {
	return t1.After(t2)
}

// IsBetween 检查时间是否在两个时间之间
func IsBetween(t, start, end time.Time) bool {
	return (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end))
}

// IsToday 检查时间是否是今天
func IsToday(t time.Time) bool {
	now := time.Now()
	year, month, day := t.Date()
	nowYear, nowMonth, nowDay := now.Date()
	
	return year == nowYear && month == nowMonth && day == nowDay
}

// IsYesterday 检查时间是否是昨天
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	year, month, day := t.Date()
	yYear, yMonth, yDay := yesterday.Date()
	
	return year == yYear && month == yMonth && day == yDay
}

// IsTomorrow 检查时间是否是明天
func IsTomorrow(t time.Time) bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	year, month, day := t.Date()
	tYear, tMonth, tDay := tomorrow.Date()
	
	return year == tYear && month == tMonth && day == tDay
}

// StartOfDay 获取当天开始时间
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取当天结束时间
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek 获取本周开始时间（周一）
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日为7
	}
	
	days := weekday - 1
	return StartOfDay(t.AddDate(0, 0, -days))
}

// EndOfWeek 获取本周结束时间（周日）
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth 获取本月开始时间
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取本月结束时间
func EndOfMonth(t time.Time) time.Time {
	firstOfMonth := StartOfMonth(t)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	return EndOfDay(lastOfMonth)
}

// StartOfYear 获取本年开始时间
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear 获取本年结束时间
func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
}

// DaysInMonth 获取指定月份的天数
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// IsLeapYear 检查是否为闰年
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// Age 计算年龄
func Age(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	
	if now.Month() < birthDate.Month() || 
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}
	
	return age
}

// TimeUntil 计算距离指定时间还有多久
func TimeUntil(target time.Time) time.Duration {
	return time.Until(target)
}

// TimeSince 计算距离指定时间过去了多久
func TimeSince(t time.Time) time.Duration {
	return time.Since(t)
}

// AddDuration 添加持续时间
func AddDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// SubDuration 减去持续时间
func SubDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(-d)
}

// Elapsed 计算经过的时间
func Elapsed(start time.Time) time.Duration {
	return time.Since(start)
}

// Timeout 创建超时检查器
func Timeout(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

// Ticker 创建定时器
func Ticker(duration time.Duration) *time.Ticker {
	return time.NewTicker(duration)
}

// Sleep 睡眠指定时间
func Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// UnixToTime 将Unix时间戳转换为时间
func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// TimeToUnix 将时间转换为Unix时间戳
func TimeToUnix(t time.Time) int64 {
	return t.Unix()
}

// UnixMilliToTime 将Unix毫秒时间戳转换为时间
func UnixMilliToTime(unixMilli int64) time.Time {
	return time.Unix(unixMilli/1000, (unixMilli%1000)*1000000)
}

// TimeToUnixMilli 将时间转换为Unix毫秒时间戳
func TimeToUnixMilli(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

// BackoffDelay 计算指数退避延迟
func BackoffDelay(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := baseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > maxDelay {
			return maxDelay
		}
	}
	return delay
}

// Jitter 添加随机抖动到延迟时间
func Jitter(duration time.Duration, factor float64) time.Duration {
	if factor <= 0 || factor > 1 {
		return duration
	}
	
	jitterAmount := float64(duration) * factor
	// 这里简化处理，实际使用时应该使用随机数
	return duration + time.Duration(jitterAmount/2)
}

// TimeRange 时间范围结构
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange 创建时间范围
func NewTimeRange(start, end time.Time) *TimeRange {
	return &TimeRange{Start: start, End: end}
}

// Duration 获取时间范围的持续时间
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains 检查时间是否在范围内
func (tr *TimeRange) Contains(t time.Time) bool {
	return IsBetween(t, tr.Start, tr.End)
}

// Overlaps 检查两个时间范围是否重叠
func (tr *TimeRange) Overlaps(other *TimeRange) bool {
	return tr.Start.Before(other.End) && tr.End.After(other.Start)
}