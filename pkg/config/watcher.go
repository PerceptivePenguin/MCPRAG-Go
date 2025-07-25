package config

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"
)

// Watcher 配置文件监视器接口
type Watcher interface {
	Start(ctx context.Context) error
	Stop() error
	OnChange(callback func(string) error)
}

// FileWatcher 文件监视器
type FileWatcher struct {
	mu        sync.RWMutex
	filePath  string
	lastMod   time.Time
	callback  func(string) error
	ticker    *time.Ticker
	stopCh    chan struct{}
	started   bool
}

// NewFileWatcher 创建文件监视器
func NewFileWatcher(filePath string, interval time.Duration) *FileWatcher {
	if interval <= 0 {
		interval = 5 * time.Second // 默认5秒检查一次
	}
	
	return &FileWatcher{
		filePath: filePath,
		ticker:   time.NewTicker(interval),
		stopCh:   make(chan struct{}),
	}
}

// Start 开始监视
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if fw.started {
		return fmt.Errorf("watcher already started")
	}
	
	// 获取初始修改时间
	if stat, err := os.Stat(fw.filePath); err == nil {
		fw.lastMod = stat.ModTime()
	}
	
	fw.started = true
	
	go fw.watch(ctx)
	return nil
}

// Stop 停止监视
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if !fw.started {
		return nil
	}
	
	fw.started = false
	close(fw.stopCh)
	fw.ticker.Stop()
	
	return nil
}

// OnChange 设置变更回调
func (fw *FileWatcher) OnChange(callback func(string) error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.callback = callback
}

// watch 监视文件变更
func (fw *FileWatcher) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-fw.stopCh:
			return
		case <-fw.ticker.C:
			fw.checkFileChange()
		}
	}
}

// checkFileChange 检查文件变更
func (fw *FileWatcher) checkFileChange() {
	stat, err := os.Stat(fw.filePath)
	if err != nil {
		// 文件不存在或无法访问
		return
	}
	
	fw.mu.RLock()
	lastMod := fw.lastMod
	callback := fw.callback
	fw.mu.RUnlock()
	
	if stat.ModTime().After(lastMod) {
		fw.mu.Lock()
		fw.lastMod = stat.ModTime()
		fw.mu.Unlock()
		
		if callback != nil {
			if err := callback(fw.filePath); err != nil {
				// TODO: 添加日志记录
				fmt.Printf("Config change callback error: %v\n", err)
			}
		}
	}
}

// MultiWatcher 多文件监视器
type MultiWatcher struct {
	mu       sync.RWMutex
	watchers []Watcher
	callback func(string) error
}

// NewMultiWatcher 创建多文件监视器
func NewMultiWatcher() *MultiWatcher {
	return &MultiWatcher{
		watchers: make([]Watcher, 0),
	}
}

// AddWatcher 添加监视器
func (mw *MultiWatcher) AddWatcher(watcher Watcher) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	
	watcher.OnChange(mw.onFileChange)
	mw.watchers = append(mw.watchers, watcher)
}

// Start 启动所有监视器
func (mw *MultiWatcher) Start(ctx context.Context) error {
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	
	for _, watcher := range mw.watchers {
		if err := watcher.Start(ctx); err != nil {
			return fmt.Errorf("failed to start watcher: %w", err)
		}
	}
	
	return nil
}

// Stop 停止所有监视器
func (mw *MultiWatcher) Stop() error {
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	
	var lastErr error
	for _, watcher := range mw.watchers {
		if err := watcher.Stop(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// OnChange 设置变更回调
func (mw *MultiWatcher) OnChange(callback func(string) error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.callback = callback
}

// onFileChange 文件变更处理
func (mw *MultiWatcher) onFileChange(filePath string) error {
	mw.mu.RLock()
	callback := mw.callback
	mw.mu.RUnlock()
	
	if callback != nil {
		return callback(filePath)
	}
	
	return nil
}

// AutoReloader 自动重载器
type AutoReloader struct {
	manager  *Manager
	watcher  Watcher
	target   interface{}
	mu       sync.RWMutex
}

// NewAutoReloader 创建自动重载器
func NewAutoReloader(manager *Manager, watcher Watcher, target interface{}) *AutoReloader {
	reloader := &AutoReloader{
		manager: manager,
		watcher: watcher,
		target:  target,
	}
	
	watcher.OnChange(reloader.reload)
	return reloader
}

// Start 启动自动重载
func (ar *AutoReloader) Start(ctx context.Context) error {
	return ar.watcher.Start(ctx)
}

// Stop 停止自动重载
func (ar *AutoReloader) Stop() error {
	return ar.watcher.Stop()
}

// reload 重新加载配置
func (ar *AutoReloader) reload(filePath string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	
	fmt.Printf("Reloading config from: %s\n", filePath)
	
	// 创建新的配置实例
	newConfig := reflect.New(reflect.TypeOf(ar.target).Elem()).Interface()
	
	// 重新加载配置
	if err := ar.manager.Load(newConfig); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	// 更新目标配置
	reflect.ValueOf(ar.target).Elem().Set(reflect.ValueOf(newConfig).Elem())
	
	fmt.Printf("Config reloaded successfully\n")
	return nil
}