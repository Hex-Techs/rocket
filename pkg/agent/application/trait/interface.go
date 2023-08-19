package trait

import (
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// EventType handler event type
type EventType string

const (
	// create or update event
	CreatedOrUpdate EventType = "createdOrUpdate"
	// delete event
	Deleted EventType = "deleted"
)

var (
	Traits = map[string]Trait{}
)

// register trait
func RegisterTrait(kind string, t Trait) {
	Traits[kind] = t
	constant.EdgeTrait.Add(kind)
}

type Trait interface {
	// 根据给定的 traitTemplate 生成相应的配置
	Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error
	// 根据 trait 进行相应的处理
	Handler(trait *rocketv1alpha1.Trait, app *rocketv1alpha1.Application,
		event EventType, client *Client) error
}

func NewClient(kclient kclientset.Interface, rclient clientset.Interface, client kubernetes.Interface) *Client {
	return &Client{
		Kclient: kclient,
		Rclient: rclient,
		Client:  client,
	}
}

type Client struct {
	Kclient kclientset.Interface
	Rclient clientset.Interface
	Client  kubernetes.Interface
}
