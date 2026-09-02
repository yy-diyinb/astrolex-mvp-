package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	currentLang  string
	translations map[string]string
	mu           sync.RWMutex
)

// Load 加载指定语言文件
func Load(lang string) error {
	mu.Lock()
	defer mu.Unlock()
	path := filepath.Join("i18n", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var t map[string]string
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	translations = t
	currentLang = lang
	return nil
}

// T 翻译 key，支持简单格式化（如 "Hello %s"）
func T(key string, args ...interface{}) string {
	mu.RLock()
	defer mu.RUnlock()
	if msg, ok := translations[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	return key // 未找到返回 key 本身
}
