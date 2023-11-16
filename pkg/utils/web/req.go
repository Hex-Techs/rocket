package web

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// 默认单页数据量
	_defaultPageSize = 20
	// 默认当前页
	_defaultCurrentPage = 1
)

// Request request object
type Request struct {
	//  页码
	Page int `form:"page"`
	// 每页数量
	Limit int `form:"limit"`
	// 父模块id
	ParentID int `form:"parentID"`
	// 模块级别
	Level int `form:"level"`
	// 项目名称
	Project string `form:"project"`
	// 集群名称
	Cluster string `form:"cluster"`
	// 命名空间
	Namespace string `form:"namespace"`
	// 资源名称
	Name string `form:"name"`
	// Workload名称
	Workload string `form:"workload"`
	// 资源owner
	Owner string `form:"owner"`
	// 资源owner类型
	OwnerKind string `form:"ownerKind"`
	// 可能会用到的参数
	Log       bool   `form:"log"`
	Event     bool   `form:"event"`
	Container string `form:"container"`
	Follow    bool   `form:"follow"`
	Tail      int    `form:"tail"`
	Previous  bool   `form:"previous"`
	SinceTime string `form:"sinceTime"`
	Describe  bool   `form:"describe"`
	// Watch     bool   `form:"watch"`
	Deleted bool `form:"deleted"`
}

func (q *Request) Default() {
	if q.Limit <= 0 {
		q.Limit = -1
		q.Page = 1
	}
}

// HandleDefult 处理分页参数
func (q *Request) HandleDefult(total int) {
	totalPages := 1
	// limit <0 时，不分页
	if q.Limit < 0 {
		q.Page = totalPages
		q.Limit = total
		return
	}
	if q.Page <= 0 {
		q.Page = _defaultCurrentPage
	}
	if q.Limit == 0 {
		q.Limit = _defaultPageSize
	}
	if total > q.Limit {
		totalPages = total / q.Limit
		if total%q.Limit > 0 {
			totalPages = totalPages + 1
		}
	}
	if q.Page > totalPages {
		q.Page = totalPages
	}
}

type Query struct {
	// cluster to form
	Cluster string
	// namespace to form
	Namespace string
	// current page
	CurrentPage int
	// page size, the limit of one page
	PageSize int
	// list option for k8s resource
	ListOption *metav1.ListOptions
	// get option for k8s resource
	GetOption *metav1.GetOptions
	// update option for k8s resource
	UpdateOption *metav1.UpdateOptions
	// raw label selector
	Selector labels.Selector
}
