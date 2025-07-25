package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ConvertUtils 数据转换工具函数

// ToString 将任意类型转换为字符串
func ToString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return strconv.FormatBool(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToInt 将任意类型转换为整数
func ToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ToInt64 将任意类型转换为int64
func ToInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

// ToFloat64 将任意类型转换为float64
func ToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ToBool 将任意类型转换为布尔值
func ToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int, int8, int16, int32, int64:
		return v != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return v != 0, nil
	case float32:
		return v != 0.0, nil
	case float64:
		return v != 0.0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// ToStringSlice 将任意类型转换为字符串切片
func ToStringSlice(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	
	v := reflect.ValueOf(value)
	
	// 如果已经是字符串切片
	if v.Type() == reflect.TypeOf([]string{}) {
		return value.([]string), nil
	}
	
	// 如果是切片或数组
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		result := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = ToString(v.Index(i).Interface())
		}
		return result, nil
	}
	
	// 如果是字符串，按逗号分割
	if s, ok := value.(string); ok {
		if strings.TrimSpace(s) == "" {
			return []string{}, nil
		}
		return SplitAndTrim(s, ","), nil
	}
	
	return nil, fmt.Errorf("cannot convert %T to []string", value)
}

// ToIntSlice 将任意类型转换为整数切片
func ToIntSlice(value interface{}) ([]int, error) {
	if value == nil {
		return nil, nil
	}
	
	v := reflect.ValueOf(value)
	
	// 如果已经是整数切片
	if v.Type() == reflect.TypeOf([]int{}) {
		return value.([]int), nil
	}
	
	// 如果是切片或数组
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		result := make([]int, v.Len())
		for i := 0; i < v.Len(); i++ {
			intVal, err := ToInt(v.Index(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("failed to convert element %d to int: %w", i, err)
			}
			result[i] = intVal
		}
		return result, nil
	}
	
	// 如果是字符串，按逗号分割并转换
	if s, ok := value.(string); ok {
		if strings.TrimSpace(s) == "" {
			return []int{}, nil
		}
		parts := SplitAndTrim(s, ",")
		result := make([]int, len(parts))
		for i, part := range parts {
			intVal, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("failed to convert '%s' to int: %w", part, err)
			}
			result[i] = intVal
		}
		return result, nil
	}
	
	return nil, fmt.Errorf("cannot convert %T to []int", value)
}

// ToMap 将任意类型转换为map[string]interface{}
func ToMap(value interface{}) (map[string]interface{}, error) {
	if value == nil {
		return nil, nil
	}
	
	// 如果已经是map[string]interface{}
	if m, ok := value.(map[string]interface{}); ok {
		return m, nil
	}
	
	// 如果是其他类型的map
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Map {
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			keyStr := ToString(key.Interface())
			result[keyStr] = v.MapIndex(key).Interface()
		}
		return result, nil
	}
	
	// 如果是结构体
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		result := make(map[string]interface{})
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)
			
			if !field.CanInterface() {
				continue
			}
			
			// 获取字段名，优先使用json tag
			fieldName := fieldType.Name
			if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					fieldName = parts[0]
				}
			}
			
			result[fieldName] = field.Interface()
		}
		return result, nil
	}
	
	return nil, fmt.Errorf("cannot convert %T to map[string]interface{}", value)
}

// ToJSON 将任意类型转换为JSON字符串
func ToJSON(value interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}
	
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	
	return string(bytes), nil
}

// FromJSON 从JSON字符串解析为指定类型
func FromJSON(jsonStr string, target interface{}) error {
	if jsonStr == "" {
		return fmt.Errorf("empty JSON string")
	}
	
	return json.Unmarshal([]byte(jsonStr), target)
}

// ToJSONPretty 将任意类型转换为格式化的JSON字符串
func ToJSONPretty(value interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}
	
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to pretty JSON: %w", err)
	}
	
	return string(bytes), nil
}

// DeepCopy 深拷贝对象（通过JSON序列化/反序列化）
func DeepCopy(src, dst interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}
	
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}
	
	return nil
}

// IsNil 检查值是否为nil
func IsNil(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// IsZeroValue 检查值是否为零值
func IsZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	return v.IsZero()
}

// Coalesce 返回第一个非nil且非零值的参数
func Coalesce(values ...interface{}) interface{} {
	for _, value := range values {
		if !IsNil(value) && !IsZeroValue(value) {
			return value
		}
	}
	return nil
}

// DefaultValue 如果值为nil或零值则返回默认值
func DefaultValue(value, defaultValue interface{}) interface{} {
	if IsNil(value) || IsZeroValue(value) {
		return defaultValue
	}
	return value
}