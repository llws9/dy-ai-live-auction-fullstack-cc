package nacos

import (
	"fmt"
	"log"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NacosClient Nacos 配置中心客户端
type NacosClient struct {
	client     config_client.IConfigClient
	namespace  string
	serverAddr string
}

// Config Nacos 连接配置
type Config struct {
	ServerAddr string
	Namespace  string
}

// NewNacosClient 创建 Nacos 客户端
func NewNacosClient(cfg *Config) (*NacosClient, error) {
	// 解析服务器地址（格式：host:port）
	host, port := parseServerAddr(cfg.ServerAddr)

	clientConfig := &constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
	TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "info",
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   port,
		},
	}

	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:   clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos client: %w", err)
	}

	log.Printf("Nacos client connected to %s, namespace: %s", cfg.ServerAddr, cfg.Namespace)

	return &NacosClient{
		client:     client,
		namespace:  cfg.Namespace,
		serverAddr: cfg.ServerAddr,
	}, nil
}

// GetConfig 获取配置内容
func (c *NacosClient) GetConfig(group, dataId string) (string, error) {
	content, err := c.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get config [group=%s, dataId=%s]: %w", group, dataId, err)
	}
	return content, nil
}

// ListenConfig 监听配置变更
func (c *NacosClient) ListenConfig(group, dataId string, onChange func(namespace, group, dataId, content string)) error {
	err := c.client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, content string) {
			log.Printf("Config changed: [namespace=%s, group=%s, dataId=%s]", namespace, group, dataId)
			if onChange != nil {
				onChange(namespace, group, dataId, content)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to listen config: %w", err)
	}
	log.Printf("Listening config: [group=%s, dataId=%s]", group, dataId)
	return nil
}

// CancelListenConfig 取消监听配置
func (c *NacosClient) CancelListenConfig(group, dataId string) error {
	return c.client.CancelListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
}

// parseServerAddr 解析服务器地址
func parseServerAddr(addr string) (string, uint64) {
	host := addr
	port := uint64(8848)

	// 如果地址包含端口，则解析
	if idx := len(addr) - 1; idx > 0 {
		for i := idx; i >= 0; i-- {
			if addr[i] == ':' {
				host = addr[:i]
				if p, err := parsePort(addr[i+1:]); err == nil {
					port = p
				}
				break
			}
		}
	}

	return host, port
}

func parsePort(s string) (uint64, error) {
	var port uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port * 10 + uint64(c-'0')
		} else {
			return 0, fmt.Errorf("invalid port")
		}
	}
	return port, nil
}