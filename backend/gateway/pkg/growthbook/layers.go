package growthbook

import (
	"fmt"
	"sync"
)

// ExperimentLayer 实验层级定义
// 用于父子实验配置，避免实验碰撞
type ExperimentLayer struct {
	Name        string
	Namespace   string
	Experiments []string // 该层级内的实验列表
}

// LayerManager 实验层级管理器
type LayerManager struct {
	layers     map[string]*ExperimentLayer
	layersLock sync.RWMutex
}

// NewLayerManager 创建层级管理器
func NewLayerManager() *LayerManager {
	return &LayerManager{
		layers: make(map[string]*ExperimentLayer),
	}
}

// AddLayer 添加实验层级
func (lm *LayerManager) AddLayer(layer *ExperimentLayer) {
	lm.layersLock.Lock()
	lm.layers[layer.Name] = layer
	lm.layersLock.Unlock()
}

// GetLayer 获取实验层级
func (lm *LayerManager) GetLayer(name string) *ExperimentLayer {
	lm.layersLock.RLock()
	layer := lm.layers[name]
	lm.layersLock.RUnlock()
	return layer
}

// CheckLayerCollision 检查实验层级碰撞
// 返回碰撞的层级名称列表
func (lm *LayerManager) CheckLayerCollision(experimentKey string) []string {
	lm.layersLock.RLock()
	defer lm.layersLock.RUnlock()

	collisions := []string{}
	for name, layer := range lm.layers {
		for _, exp := range layer.Experiments {
			if exp == experimentKey {
				collisions = append(collisions, name)
			}
		}
	}
	return collisions
}

// EvalLayeredFeature 评估层级实验
// 先检查父实验，再评估子实验
func (lm *LayerManager) EvalLayeredFeature(client *Client, parentKey string, childKey string, attrs *Attributes) (*EvalResult, *EvalResult) {
	// 先评估父实验
	parentResult := client.EvalFeature(parentKey, attrs)

	// 如果父实验开启，再评估子实验
	if parentResult.On {
		childResult := client.EvalFeature(childKey, attrs)
		return parentResult, childResult
	}

	return parentResult, nil
}

// DefaultLayers 默认实验层级配置
func DefaultLayers() *LayerManager {
	lm := NewLayerManager()

	// UI 层实验
	lm.AddLayer(&ExperimentLayer{
		Name:      "ui-layer",
		Namespace: "ui",
		Experiments: []string{
			"new-auction-ui-theme",
			"bid-button-color",
			"admin-ui-style",
		},
	})

	// 业务层实验
	lm.AddLayer(&ExperimentLayer{
		Name:      "business-layer",
		Namespace: "business",
		Experiments: []string{
			"new-bidding-algorithm",
			"price-suggestion-strategy",
			"auction-sorting",
		},
	})

	return lm
}

// ParentChildExperiment 父子实验关系定义
type ParentChildExperiment struct {
	ParentKey string
	ChildKeys []string
}

// ValidateParentChild 验证父子实验关系
func (lm *LayerManager) ValidateParentChild(pc *ParentChildExperiment) error {
	parentLayer := lm.GetLayer(pc.ParentKey)
	if parentLayer == nil {
		return fmt.Errorf("parent experiment %s not found in any layer", pc.ParentKey)
	}

	for _, childKey := range pc.ChildKeys {
		childCollisions := lm.CheckLayerCollision(childKey)
		if len(childCollisions) > 0 {
			return fmt.Errorf("child experiment %s has collision with layer %s", childKey, childCollisions[0])
		}
	}

	return nil
}