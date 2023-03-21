package constant

// 用于设置多可用区的节点的label
const TopologyZoneLabel = "topology.kubernetes.io/zone"

const (
	// 用于设置使用了哪些template的label
	TemplateUsedLabel = "rocket.hextech.io/templates"
	// 用于设置workload的label
	WorkloadNameLabel = "rocket.hextech.io/workload"
	// 用于设置app的label
	AppNameLabel = "rocket.hextech.io/app"
	// 用于设置workload通过那个app生成的
	GenerateNameLabel = "rocket.hextech.io/generate-name"
	// rocket的管理label
	ManagedByRocketLabel = "app.kubernetes.io/managed-by"
	// 用于设置cloudarea和region的label和环境信息
	CloudAreaLabel = "rocket.hextech.io/cloud_area"
	RegionLabel    = "rocket.hextech.io/region"
	EnvLabel       = "rocket.hextech.io/env"
)
