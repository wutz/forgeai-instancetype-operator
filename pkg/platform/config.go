// Package platform reads the part of the forgeai platform config that the
// InstanceType operator needs: how GPU nodes are sliced.
//
// The operator reads the same ConfigMap as forgeai (forgeai-system/forgeai-platform,
// key config.yaml) and ignores every other key.
package platform

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	// ConfigMapName holds the platform config under key ConfigKey.
	ConfigMapName = "forgeai-platform"
	ConfigKey     = "config.yaml"
	// DefaultNamespace is where forgeai and the operator live.
	DefaultNamespace = "forgeai-system"
)

// Config is the subset of the platform config the operator reads.
type Config struct {
	InstanceTypes InstanceTypes `json:"instanceTypes"`
}

// InstanceTypes controls how GPU nodes are sliced.
type InstanceTypes struct {
	// Reserved is kept back on every node for the system before slicing.
	Reserved struct {
		CPU    resource.Quantity `json:"cpu"`
		Memory resource.Quantity `json:"memory"`
	} `json:"reserved"`
	// Sizes are the GPU counts offered (capped at the GPUs per node).
	Sizes []int32 `json:"sizes,omitempty"`
}

// Parse parses YAML and applies defaults.
func Parse(data []byte) (*Config, error) {
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse platform config: %w", err)
	}
	if len(c.InstanceTypes.Sizes) == 0 {
		c.InstanceTypes.Sizes = []int32{1, 2, 4, 8}
	}
	defaultQuantity(&c.InstanceTypes.Reserved.CPU, "2")
	defaultQuantity(&c.InstanceTypes.Reserved.Memory, "16Gi")
	return c, nil
}

func defaultQuantity(q *resource.Quantity, v string) {
	if q.IsZero() {
		*q = resource.MustParse(v)
	}
}

// Namespace returns the platform namespace from FORGEAI_NAMESPACE or the default.
func Namespace() string {
	if ns := os.Getenv("FORGEAI_NAMESPACE"); ns != "" {
		return ns
	}
	return DefaultNamespace
}

// Load reads the platform ConfigMap.
func Load(ctx context.Context, c client.Reader, namespace string) (*Config, error) {
	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ConfigMapName}, cm); err != nil {
		return nil, fmt.Errorf("get platform config %s/%s: %w", namespace, ConfigMapName, err)
	}
	return Parse([]byte(cm.Data[ConfigKey]))
}

// Loader caches the platform config for a short time.
type Loader struct {
	Reader    client.Reader
	Namespace string

	mu   sync.Mutex
	cfg  *Config
	time time.Time
}

// Get returns the cached config, reloading it after 15 seconds.
func (l *Loader) Get(ctx context.Context) (*Config, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cfg != nil && time.Since(l.time) < 15*time.Second {
		return l.cfg, nil
	}
	cfg, err := Load(ctx, l.Reader, l.Namespace)
	if err != nil {
		return nil, err
	}
	l.cfg, l.time = cfg, time.Now()
	return cfg, nil
}

// Invalidate drops the cached config, so that the next Get reads the
// ConfigMap again.
func (l *Loader) Invalidate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg = nil
}
