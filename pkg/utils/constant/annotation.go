package constant

const (
	// 触发强制刷新，用在template修改后是否触发app的更改
	FlushAnnotation = "rocket.hextech.io/flush"

	// template资源使用的annaotation
	TemplateUsed = "rocket.hextech.io/templates"

	// 最后一次调度的集群
	LastSchedulerClusterAnnotation = "rocket.hextech.io/last-scheduler-cluster"

	// trait的edge类型的名称
	TraitEdgeAnnotation = "rocket.hextech.io/edge"
)

// ExtendedResourceAnnotation if set this annotation to true, then set
// CloneSet as primary workload, if not set or set to false, then set
// Deplooyment as primary workload.
const (
	ExtendedResourceAnnotation = "rocket.hextech.io/extended-resource"
)
