// Package config 提供了统一的配置管理系统
//
// 这个包提供了配置加载、验证、合并和环境变量支持，包括：
// - 多种配置源支持（文件、环境变量、默认值）
// - 配置验证和类型检查
// - 配置热重载支持
// - 敏感信息处理
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Loader 配置加载器接口
type Loader interface {
	Load(target interface{}) error
	Validate(config interface{}) error
}

// Manager 配置管理器
type Manager struct {
	mu       sync.RWMutex
	loaders  []Loader
	watchers []Watcher
	config   interface{}
}

// NewManager 创建新的配置管理器
func NewManager() *Manager {
	return &Manager{
		loaders:  make([]Loader, 0),
		watchers: make([]Watcher, 0),
	}
}

// AddLoader 添加配置加载器
func (m *Manager) AddLoader(loader Loader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loaders = append(m.loaders, loader)
}

// Load 加载配置
func (m *Manager) Load(target interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, loader := range m.loaders {
		if err := loader.Load(target); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	
	// 验证配置
	for _, loader := range m.loaders {
		if err := loader.Validate(target); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}
	
	m.config = target
	return nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// FileLoader 文件配置加载器
type FileLoader struct {
	FilePath string
	Format   string // json, yaml, yml
}

// NewFileLoader 创建文件配置加载器
func NewFileLoader(filePath string) *FileLoader {
	format := strings.ToLower(filepath.Ext(filePath))
	if format != "" {
		format = format[1:] // 移除点号
	}
	
	return &FileLoader{
		FilePath: filePath,
		Format:   format,
	}
}

// Load 从文件加载配置
func (fl *FileLoader) Load(target interface{}) error {
	if _, err := os.Stat(fl.FilePath); os.IsNotExist(err) {
		// 文件不存在时不报错，允许使用默认配置
		return nil
	}
	
	data, err := os.ReadFile(fl.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", fl.FilePath, err)
	}
	
	switch fl.Format {
	case "json":
		return json.Unmarshal(data, target)
	case "yaml", "yml":
		return yaml.Unmarshal(data, target)
	default:
		return fmt.Errorf("unsupported config format: %s", fl.Format)
	}
}

// Validate 验证配置
func (fl *FileLoader) Validate(config interface{}) error {
	// 基础验证，检查必需字段
	return validateStruct(config)
}

// EnvLoader 环境变量配置加载器
type EnvLoader struct {
	Prefix string // 环境变量前缀
}

// NewEnvLoader 创建环境变量配置加载器
func NewEnvLoader(prefix string) *EnvLoader {
	return &EnvLoader{
		Prefix: prefix,
	}
}

// Load 从环境变量加载配置
func (el *EnvLoader) Load(target interface{}) error {
	return loadFromEnv(target, el.Prefix)
}

// Validate 验证配置
func (el *EnvLoader) Validate(config interface{}) error {
	return validateStruct(config)
}

// DefaultLoader 默认值配置加载器
type DefaultLoader struct {
	DefaultConfig interface{}
}

// NewDefaultLoader 创建默认值配置加载器
func NewDefaultLoader(defaultConfig interface{}) *DefaultLoader {
	return &DefaultLoader{
		DefaultConfig: defaultConfig,
	}
}

// Load 加载默认配置
func (dl *DefaultLoader) Load(target interface{}) error {
	return copyDefaults(dl.DefaultConfig, target)
}

// Validate 验证配置
func (dl *DefaultLoader) Validate(config interface{}) error {
	return validateStruct(config)
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(target interface{}, prefix string) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}
	
	v = v.Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if !field.CanSet() {
			continue
		}
		
		// 获取环境变量名
		envKey := getEnvKey(fieldType, prefix)
		if envKey == "" {
			continue
		}
		
		envValue := os.Getenv(envKey)
		if envValue == "" {
			continue
		}
		
		// 设置字段值
		if err := setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, envKey, err)
		}
	}
	
	return nil
}

// getEnvKey 获取环境变量键名
func getEnvKey(field reflect.StructField, prefix string) string {
	// 检查env tag
	if envTag := field.Tag.Get("env"); envTag != "" {
		if prefix != "" {
			return prefix + "_" + envTag
		}
		return envTag
	}
	
	// 检查json tag
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" && parts[0] != "-" {
			key := strings.ToUpper(strings.ReplaceAll(parts[0], "_", "_"))
			if prefix != "" {
				return prefix + "_" + key
			}
			return key
		}
	}
	
	// 使用字段名
	key := strings.ToUpper(field.Name)
	if prefix != "" {
		return prefix + "_" + key
	}
	return key
}

// setFieldValue 设置字段值
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// 特殊处理time.Duration
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	
	return nil
}

// copyDefaults 复制默认值
func copyDefaults(src, dst interface{}) error {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)
	
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct")
	}
	
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	
	if srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("src must be a struct or pointer to struct")
	}
	
	dstVal = dstVal.Elem()
	srcType := srcVal.Type()
	dstType := dstVal.Type()
	
	if srcType != dstType {
		return fmt.Errorf("src and dst must be the same type")
	}
	
	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		dstField := dstVal.Field(i)
		
		if !dstField.CanSet() {
			continue
		}
		
		// 如果目标字段为零值，则使用默认值
		if isZeroValue(dstField) {
			dstField.Set(srcField)
		}
	}
	
	return nil
}

// isZeroValue 检查是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// validateStruct 验证结构体
func validateStruct(config interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// 检查required tag
		if required := fieldType.Tag.Get("required"); required == "true" {
			if isZeroValue(field) {
				return fmt.Errorf("required field %s is missing", fieldType.Name)
			}
		}
		
		// 检查validate tag
		if validate := fieldType.Tag.Get("validate"); validate != "" {
			if err := validateField(field, validate, fieldType.Name); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateField 验证字段
func validateField(field reflect.Value, rule, fieldName string) error {
	rules := strings.Split(rule, ",")
	
	for _, r := range rules {
		r = strings.TrimSpace(r)
		
		if strings.HasPrefix(r, "min=") {
			minStr := strings.TrimPrefix(r, "min=")
			min, err := strconv.Atoi(minStr)
			if err != nil {
				return fmt.Errorf("invalid min rule for field %s: %s", fieldName, minStr)
			}
			
			switch field.Kind() {
			case reflect.String:
				if field.Len() < min {
					return fmt.Errorf("field %s must be at least %d characters", fieldName, min)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if field.Int() < int64(min) {
					return fmt.Errorf("field %s must be at least %d", fieldName, min)
				}
			}
		}
		
		if strings.HasPrefix(r, "max=") {
			maxStr := strings.TrimPrefix(r, "max=")
			max, err := strconv.Atoi(maxStr)
			if err != nil {
				return fmt.Errorf("invalid max rule for field %s: %s", fieldName, maxStr)
			}
			
			switch field.Kind() {
			case reflect.String:
				if field.Len() > max {
					return fmt.Errorf("field %s must be at most %d characters", fieldName, max)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if field.Int() > int64(max) {
					return fmt.Errorf("field %s must be at most %d", fieldName, max)
				}
			}
		}
	}
	
	return nil
}