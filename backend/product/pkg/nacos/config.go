package nacos

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	client    *NacosClient
	group     string
	dataId    string
	config    interface{}
	mu        sync.RWMutex
	onChange  func(interface{})
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(client *NacosClient, group, dataId string) *ConfigLoader {
	return &ConfigLoader{
		client: client,
		group:  group,
		dataId: dataId,
	}
}

// Load 加载配置到目标结构体
func (l *ConfigLoader) Load(target interface{}) error {
	content, err := l.client.GetConfig(l.group, l.dataId)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := yaml.Unmarshal([]byte(content), target); err != nil {
		return fmt.Errorf("failed to parse config yaml: %w", err)
	}

	l.mu.Lock()
	l.config = target
	l.mu.Unlock()

	log.Printf("Config loaded from Nacos: [group=%s, dataId=%s]", l.group, l.dataId)
	return nil
}

// LoadAndListen 加载配置并监听变更
func (l *ConfigLoader) LoadAndListen(target interface{}, onChange func(interface{})) error {
	// 先加载初始配置
	if err := l.Load(target); err != nil {
		return err
	}

	l.onChange = onChange

	// 监听配置变更
	err := l.client.ListenConfig(l.group, l.dataId, func(namespace, group, dataId, content string) {
		l.mu.Lock()
		if err := yaml.Unmarshal([]byte(content), target); err != nil {
			log.Printf("Failed to parse updated config: %v", err)
			l.mu.Unlock()
			return
		}
		l.config = target
		l.mu.Unlock()

		log.Printf("Config updated from Nacos: [group=%s, dataId=%s]", group, dataId)
		if l.onChange != nil {
			l.onChange(target)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to listen config changes: %w", err)
	}

	return nil
}

// GetConfig 从环境变量加载 Nacos 配置
func GetConfigFromEnv() *Config {
	return &Config{
		ServerAddr: getEnvOrDefault("NACOS_SERVER_ADDR", "localhost:8848"),
		Namespace:  getEnvOrDefault("NACOS_NAMESPACE", "auction-dev"),
	}
}

// GetServiceConfigInfo 从环境变量获取服务配置信息
func GetServiceConfigInfo() (group, dataId string) {
	group = getEnvOrDefault("NACOS_GROUP", "default")
	dataId = getEnvOrDefault("NACOS_DATA_ID", "product-config.yaml")
	return group, dataId
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
