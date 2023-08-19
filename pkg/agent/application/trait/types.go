package trait

import (
	"k8s.io/apimachinery/pkg/util/intstr"
)

// register trait
func init() {
	RegisterTrait(ServiceKind, NewServiceTrait())
	RegisterTrait(PodUnavailableBudgetKind, NewPubTrait())
}

// pod 干扰预算配置
type PodUnavailableBudget struct {
	// 最大不可用数量
	// maxUnavailable 和 minAvailable 互斥，maxUnavailable 优先生效
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// 最小可用数量
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
}

// k8s service 配置
type Service struct {
	// 是否是无头服务
	Headless bool `json:"headless,omitempty"`
	// 端口列表
	Ports []Port `json:"ports,omitempty"`
}

// service 端口配置
type Port struct {
	// 端口名
	Name string `json:"name,omitempty"`
	// 端口号
	Port int `json:"port,omitempty"`
	// 目标端口
	TargetPort intstr.IntOrString `json:"targetPort,omitempty"`
}
