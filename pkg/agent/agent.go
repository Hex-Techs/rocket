package agent

import (
	"fmt"
	"time"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	informers "github.com/hex-techs/rocket/client/informers/externalversions"
	"github.com/hex-techs/rocket/pkg/agent/application"
	"github.com/hex-techs/rocket/pkg/agent/cloneset"
	"github.com/hex-techs/rocket/pkg/agent/cluster"
	"github.com/hex-techs/rocket/pkg/agent/cronjob"
	"github.com/hex-techs/rocket/pkg/agent/deployment"
	"github.com/hex-techs/rocket/pkg/agent/distribution"
	"github.com/hex-techs/rocket/pkg/agent/workload"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/signals"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	kinformers "github.com/openkruise/kruise-api/client/informers/externalversions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const ErrTemplateSchemeNotSupported = "scheme %s is not supported yet"

type ReconcilerSetupFunc func(manager manager.Manager) error

var supportedSchemeReconciler = map[string]ReconcilerSetupFunc{
	cluster.ClusterKind: func(mgr manager.Manager) error {
		return cluster.NewReconciler(mgr).SetupWithManager(mgr)
	},
	deployment.DeploymentKind: func(mgr manager.Manager) error {
		return deployment.NewDeploymentReconciler(mgr).SetupWithManager(mgr)
	},
	cloneset.CloneSetKind: func(mgr manager.Manager) error {
		return cloneset.NewCloneSetReconciler(mgr).SetupWithManager(mgr)
	},
	cronjob.CronJobKind: func(mgr manager.Manager) error {
		return cronjob.NewCronJobReconcile(mgr).SetupWithManager(mgr)
	},
}

func InitReconcilers(mgr manager.Manager, enables []string) error {
	klog.V(4).Infof("enable controllers: %v", enables)
	for _, enable := range enables {
		if m, support := supportedSchemeReconciler[enable]; support {
			if err := m(mgr); err != nil {
				return err
			}
		} else {
			return fmt.Errorf(ErrTemplateSchemeNotSupported, enable)
		}
	}
	return nil
}

func RemoteController(mgr manager.Manager) error {
	for {
		if cluster.State == string(rocketv1alpha1.Approve) {
			break
		} else {
			klog.V(1).Info("wait for cluster approved to start remote controller")
			time.Sleep(10 * time.Second)
		}
	}
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandlerOnce()

	cfg, err := clustertools.GenerateKubeConfigFromToken(config.Pread().MasterURL,
		config.Pread().BootstrapToken, nil, 2)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// current cluster clientset
	kubeClient, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	// current cluster dynamic clientset
	dynamicClient, err := dynamic.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// current cluster discovery clientset
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// manager cluster clientset
	rocketclientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building rocket clientset: %s", err.Error())
	}

	kruiseClient, err := kclientset.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		klog.Fatalf("Error building open-kruise clientset: %s", err.Error())
	}

	// kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	rocketInformerFactory := informers.NewSharedInformerFactory(rocketclientset, time.Second*60)
	kruiseInformerFactory := kinformers.NewSharedInformerFactory(kruiseClient, time.Second*60)

	workloadcontroller := workload.NewController(dynamicClient, rocketclientset,
		rocketInformerFactory.Rocket().V1alpha1().Workloads())

	appcontroller := application.NewController(kubeClient, kruiseClient, rocketclientset,
		rocketInformerFactory.Rocket().V1alpha1().Applications())

	distributioncontroller := distribution.NewController(dynamicClient, discoveryClient,
		rocketclientset, rocketInformerFactory.Rocket().V1alpha1().Distributions())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	rocketInformerFactory.Start(stopCh)
	kruiseInformerFactory.Start(stopCh)

	go func() {
		if err = appcontroller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running application controller: %s", err.Error())
		}
	}()

	go func() {
		if err = distributioncontroller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running distribution controller: %s", err.Error())
		}
	}()

	return workloadcontroller.Run(2, stopCh)
}
