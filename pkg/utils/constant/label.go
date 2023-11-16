package constant

// 用于设置多可用区的节点的label
const TopologyZoneLabel = "topology.kubernetes.io/zone"

const (
	// 用于设置app的label
	AppNameLabel = "rocket.hextech.io/app"
	// application id
	AppIDLabel = "rocket.hextech.io/app-id"
	// rocket的管理label
	ManagedByRocketLabel = "app.kubernetes.io/managed-by"
	ProjectLabel         = "rocket.hextech.io/project"
	ProjectIDLabel       = "rocket.hextech.io/project-id"
)
