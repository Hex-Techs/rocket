package manager

import (
	"fmt"
	"sync"

	"github.com/hex-techs/rocket/pkg/manager/application"
	"github.com/hex-techs/rocket/pkg/manager/cluster"
	"github.com/hex-techs/rocket/pkg/manager/template"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type SetupWithManagerFunc func(mgr ctrl.Manager) error

var (
	// 控制器
	controllers = map[string]SetupWithManagerFunc{}

	lock = new(sync.Mutex)
)

func Init(mgr ctrl.Manager) {
	lock.Lock()
	// for _, v := range config.Pread().Sidecars {
	// 	applicationconfiguration.Sidecars.Add(v)
	// }
	lock.Unlock()
	controllers = map[string]SetupWithManagerFunc{
		"cluster":     cluster.NewRecociler(mgr).SetupWithManager,
		"template":    template.NewRecociler(mgr).SetupWithManager,
		"application": application.NewRecociler(mgr).SetupWithManager,
		// "resourcedistribution":     resourcedistribution.NewReconcile(mgr).SetupWithManager,
	}
}

// InitController setup manager, can call only one in init process.
func InitController(mgr manager.Manager, enableControllers []string) error {
	return setupController(mgr, enableControllers)
}

func setupController(mgr manager.Manager, enableControllers []string) error {
	if len(enableControllers) == 0 {
		for c := range controllers {
			enableControllers = append(enableControllers, c)
		}
	}
	for _, d := range enableControllers {
		if err := setup(d, mgr); err != nil {
			return err
		}
	}
	return nil
}

func setup(c string, mgr manager.Manager) error {
	if set, ok := controllers[c]; !ok {
		return fmt.Errorf("unknown controller %s", c)
	} else {
		klog.V(0).Infof("start '%s' controller", c)
		if err := set(mgr); err != nil {
			return err
		}
	}
	return nil
}
