package trait

import (
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/util/constant"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

var (
	Traits = map[string]Trait{}
)

func init() {
	RegisterTrait(PodUnavailableBudgetKind, NewPubTrait())
	constant.EdgeTrait.Insert(PodUnavailableBudgetKind)
}

// trait 的处理事件
type EventType string

const (
	// 创建事件
	Created EventType = "created"
	// 删除事件
	Deleted EventType = "deleted"
)

// 注册 trait
func RegisterTrait(kind string, t Trait) {
	Traits[kind] = t
}

type Trait interface {
	// 根据给定的 traitTemplate 生成相应的配置
	Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error
	// 根据 trait 进行相应的处理
	Handler(trait *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload,
		event EventType, client *Client) error
}

func NewClient(kclient kclientset.Interface, rclient clientset.Interface, client kubernetes.Interface) *Client {
	return &Client{
		kclient: kclient,
		rclient: rclient,
		client:  client,
	}
}

type Client struct {
	kclient kclientset.Interface
	rclient clientset.Interface
	client  kubernetes.Interface
}
