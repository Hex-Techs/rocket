package config

type ModuleParams struct {
	Kruise *OpenKruise
	Keda   *Keda
}

// OpenKruise kruise 相关的 helm value 配置
type OpenKruise struct {
	Enabled bool `json:"enable,omitempty"`
	// 开启哪些特性门控
	FeatureGates string `json:"featureGates,omitempty"`
	// 镜像仓库
	Repository string `json:"repository,omitempty"`
	// 镜像版本号
	Tag string `json:"tag,omitempty"`
	// request cpu
	RequestCPU string `json:"requestCPU,omitempty"`
	// limit cpu
	LimitCPU string `json:"limitCPU,omitempty"`
	// request memroy
	RequestMem string `json:"requestMem,omitempty"`
	// limit memory
	LimitMem string `json:"limitMem,omitempty"`
}

// Keda keda相关的 helm value 配置
type Keda struct {
	Enabled bool `json:"enable,omitempty"`
	// 镜像仓库
	Repository string `json:"repository,omitempty"`
	// 镜像版本号
	Tag string `json:"tag,omitempty"`
	// metric-server 镜像仓库
	MetricRepository string `json:"metricRepository,omitempty"`
	// metric-server 镜像版本
	MetricTag string `json:"metricTag,omitempty"`
}

// HelmRepo helm 仓库配置
type HelmRepo struct {
	// 指定的名称
	Name string
	// 仓库地址
	URL string
	// 用户名
	User string
	// 密码
	Password string
}

// 该仓库是否需要认证
func (h *HelmRepo) NeedAuth() bool {
	return h.User != "" && h.Password != ""
}
