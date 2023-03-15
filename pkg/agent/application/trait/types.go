package trait

import "k8s.io/apimachinery/pkg/util/intstr"

// pod 干扰预算配置
type PodUnavailableBudget struct {
	// 最大不可用数量
	// maxUnavailable 和 minAvailable 互斥，maxUnavailable 优先生效
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// 最小可用数量
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
}
