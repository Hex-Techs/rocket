package constant

const (
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
