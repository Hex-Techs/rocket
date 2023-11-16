package application

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type TraitKind string

const (
	ScaleKind     TraitKind = "scale"
	AffinityKind  TraitKind = "affinity"
	TolerateKind  TraitKind = "tolerate"
	MetricsKind   TraitKind = "metrics"
	ProbeKind     TraitKind = "probe"
	PodBudgetKind TraitKind = "podBudget"
	ServiceKind   TraitKind = "service"
)

type TraitType string

const (
	Local TraitType = "local"
	Edge  TraitType = "edge"
)

type TraitTemplate struct {
	Type     TraitType `json:"type,omitempty"`
	Kind     TraitKind `json:"kind,omitempty"`
	Version  string    `json:"version,omitempty"`
	Template string    `json:"template,omitempty"`
}

// maunal or auto
type Scale struct {
	Replicas *int32 `json:"replicas,omitempty"`
}

// 亲和性配置
type Affinity struct {
	Enable          bool                `json:"enable,omitempty"`
	NodeAffinity    *v1.NodeAffinity    `json:"nodeAffinity,omitempty"`
	PodAffinity     *v1.PodAffinity     `json:"podAffinity,omitempty"`
	PodAntiAffinity *v1.PodAntiAffinity `json:"podAntiAffinity,omitempty"`
}

// 容忍配置
type Tolerate struct {
	Enable      bool            `json:"enable,omitempty"`
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// metrics metrics 收集
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
	Enable bool `json:"enable,omitempty"`
	// Use this probe judge the container is live.
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// Use this probe judge the container can be accept request.
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// Use this probe judge thsi container start success, this checker is
	// success, livenessProbe and readinessProbe will work.
	StartupProbe *v1.Probe `json:"startupProbe,omitempty"`
}

// pod pub
type PodUnavailableBudget struct {
	Enable bool `json:"enable,omitempty"`
	// 最大不可用数量
	// maxUnavailable 和 minAvailable 互斥，maxUnavailable 优先生效
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// 最小可用数量
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
}

// k8s service 配置
type Service struct {
	Enable bool `json:"enable,omitempty"`
	// 是否是无头服务
	Headless bool `json:"headless,omitempty"`
	// 端口列表
	Ports []v1.ServicePort `json:"ports,omitempty"`
}
