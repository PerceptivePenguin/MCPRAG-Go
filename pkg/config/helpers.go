package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigHelper 配置助手函数

// LoadFromFile 从文件加载配置
func LoadFromFile[T any](filePath string, target *T) error {
	manager := NewManager()
	manager.AddLoader(NewFileLoader(filePath))
	return manager.Load(target)
}

// LoadFromFileWithEnv 从文件和环境变量加载配置
func LoadFromFileWithEnv[T any](filePath string, envPrefix string, target *T) error {
	manager := NewManager()
	
	// 先加载默认值
	if defaultConfig, ok := any(*target).(interface{ Default() T }); ok {
		manager.AddLoader(NewDefaultLoader(defaultConfig.Default()))
	}
	
	// 然后加载文件配置
	manager.AddLoader(NewFileLoader(filePath))
	
	// 最后加载环境变量配置
	manager.AddLoader(NewEnvLoader(envPrefix))
	
	return manager.Load(target)
}

// LoadFromMultipleSources 从多个源加载配置
func LoadFromMultipleSources[T any](sources []string, envPrefix string, target *T) error {
	manager := NewManager()
	
	// 加载默认值
	if defaultConfig, ok := any(*target).(interface{ Default() T }); ok {
		manager.AddLoader(NewDefaultLoader(defaultConfig.Default()))
	}
	
	// 加载多个配置文件
	for _, source := range sources {
		if fileExists(source) {
			manager.AddLoader(NewFileLoader(source))
		}
	}
	
	// 加载环境变量
	if envPrefix != "" {
		manager.AddLoader(NewEnvLoader(envPrefix))
	}
	
	return manager.Load(target)
}

// SaveToFile 保存配置到文件
func SaveToFile[T any](config T, filePath string) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// 根据文件扩展名确定格式
	ext := strings.ToLower(filepath.Ext(filePath))
	
	var data []byte
	var err error
	
	switch ext {
	case ".json":
		data, err = jsonMarshalIndent(config, "", "  ")
	case ".yaml", ".yml":
		data, err = yamlMarshal(config)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return os.WriteFile(filePath, data, 0644)
}

// FindConfigFile 查找配置文件
func FindConfigFile(baseName string, searchPaths []string) string {
	extensions := []string{".yaml", ".yml", ".json"}
	
	for _, path := range searchPaths {
		for _, ext := range extensions {
			fullPath := filepath.Join(path, baseName+ext)
			if fileExists(fullPath) {
				return fullPath
			}
		}
	}
	
	return ""
}

// GetDefaultSearchPaths 获取默认的配置文件搜索路径
func GetDefaultSearchPaths() []string {
	paths := []string{
		".", // 当前目录
		"./config", // config目录
		"./configs", // configs目录
	}
	
	// 添加用户home目录
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config"))
	}
	
	// 添加系统配置目录
	paths = append(paths, "/etc")
	
	return paths
}

// MergeConfigs 合并多个配置对象
func MergeConfigs[T any](base T, overrides ...T) T {
	result := base
	
	for _, override := range overrides {
		// 使用反射或JSON序列化来合并配置
		// 这里简化处理，实际实现可能需要更复杂的逻辑
		if err := mergeStructs(&result, override); err != nil {
			// 记录错误但不中断处理
			fmt.Printf("Warning: failed to merge config: %v\n", err)
		}
	}
	
	return result
}

// GetEnvWithDefault 获取环境变量，如果不存在则返回默认值
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetEnvDefaults 设置环境变量默认值
func SetEnvDefaults(defaults map[string]string) {
	for key, value := range defaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// LoadConfigWithWatch 加载配置并监听文件变化
func LoadConfigWithWatch[T any](filePath string, envPrefix string, target *T, onChange func(*T) error) error {
	// 首次加载配置
	if err := LoadFromFileWithEnv(filePath, envPrefix, target); err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}
	
	// 如果文件存在，设置文件监听
	if fileExists(filePath) {
		watcher := NewFileWatcher(filePath, 5*time.Second) // 5秒检查一次
		
		watcher.OnChange(func(changedFile string) error {
			fmt.Printf("Config file changed: %s, reloading...\n", changedFile)
			
			newConfig := new(T)
			if err := LoadFromFileWithEnv(filePath, envPrefix, newConfig); err != nil {
				return fmt.Errorf("failed to reload config: %w", err)
			}
			
			// 更新目标配置
			*target = *newConfig
			
			// 调用变更回调
			if onChange != nil {
				return onChange(target)
			}
			
			return nil
		})
		
		// 在新的goroutine中启动监听
		go func() {
			if err := watcher.Start(context.Background()); err != nil {
				fmt.Printf("Failed to start config watcher: %v\n", err)
			}
		}()
	}
	
	return nil
}

// ValidateConfig 验证配置
func ValidateConfig[T any](config T) error {
	// 如果配置实现了Validator接口，调用验证方法
	if validator, ok := any(config).(interface{ Validate() error }); ok {
		return validator.Validate()
	}
	
	// 使用通用验证
	return validateStruct(config)
}

// 内部辅助函数

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func mergeStructs[T any](dst *T, src T) error {
	// 简化的合并实现
	// 实际应用中可能需要使用reflect包来处理复杂的结构体合并
	return copyDefaults(src, dst)
}

// 引入外部库的包装函数
func jsonMarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func yamlMarshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}