package trait

import (
	v1 "k8s.io/api/core/v1"
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
	Ports []v1.ServicePort `json:"ports,omitempty"`
}
