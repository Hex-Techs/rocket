package trait

import (
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
)

var (
	Traits = map[string]Trait{}
)

func init() {
	RegisterTrait(AffinityKind, NewAffiTrait())
	RegisterTrait(ManualScaleKind, NewMSTrait())
	RegisterTrait(UpdateStrategyKind, NewUSTrait())
	RegisterTrait(TolerateKind, NewTolerateTrait())
	RegisterTrait(MetricsKind, NewMetricsTrait())
	RegisterTrait(ProbeKind, NewProbeTrait())
	RegisterTrait(DeletionProtectionKind, NewDPTrait())
}

// 注册 trait
func RegisterTrait(kind string, t Trait) {
	Traits[kind] = t
}

type Trait interface {
	// 根据给定的 trait 生成相应的配置
	Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error
	// 根据 trait 进行相应的处理
	Handler(trait *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error)
}
