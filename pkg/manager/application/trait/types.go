package trait

import v1 "k8s.io/api/core/v1"

// ManualScale 手动伸缩 trait 的配置格式
type ManualScale struct {
	Replicas      *int32         `json:"replicas,omitempty"`
	ScaleStrategy *ScaleStrategy `json:"scaleStrategy,omitempty"`
}

// ScaleStrategy 调度策略中删除指定pod，只在cloneset中生效
type ScaleStrategy struct {
	PodsToDelete []string `json:"podsToDelete,omitempty"`
}

// Affinity 亲和性配置
type Affinity struct {
	NodeAffinity    *v1.NodeAffinity    `json:"nodeAffinity,omitempty"`
	PodAffinity     *v1.PodAffinity     `json:"podAffinity,omitempty"`
	PodAntiAffinity *v1.PodAntiAffinity `json:"podAntiAffinity,omitempty"`
}

// Tolerate 容忍配置
type Tolerate struct {
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// DeletionProtection 资源删除保护配置
type DeletionProtection struct {
	// 对应的标签 policy.kruise.io/delete-protection
	// Always：这个对象禁止被删除，除非上述 label 被去掉
	// Cascading：这个对象如果还有可用的下属资源，则禁止被删除
	Type string `json:"type,omitempty"`
}

// Metrics metrics 收集
type Metrics struct {
	// prometheus.io/scrape: "false"
	Enable bool `json:"enable,omitempty"`
	// prometheus.io/path: /metrics
	Path string `json:"path,omitempty"`
	// prometheus.io/port: "8090"
	Port int `json:"port,omitempty"`
}

// Probe health check trait
type Probe struct {
	// Use this probe judge the container is live.
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// Use this probe judge the container can be accept request.
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// Use this probe judge thsi container start success, this checker is
	// success, livenessProbe and readinessProbe will work.
	StartupProbe *v1.Probe `json:"startupProbe,omitempty"`
}
