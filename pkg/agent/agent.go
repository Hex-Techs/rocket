package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/application"
	"github.com/hex-techs/rocket/pkg/agent/cloneset"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	"github.com/hex-techs/rocket/pkg/agent/cronjob"
	"github.com/hex-techs/rocket/pkg/agent/deployment"
	"github.com/hex-techs/rocket/pkg/agent/edgecluster"
	"github.com/hex-techs/rocket/pkg/agent/hubcluster"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const ErrTemplateSchemeNotSupported = "scheme %s is not supported yet"

type ReconcilerSetupFunc func(manager manager.Manager) error

var supportedSchemeReconciler = map[string]ReconcilerSetupFunc{
	deployment.DeploymentKind: func(mgr manager.Manager) error {
		return deployment.NewDeploymentReconciler(mgr).SetupWithManager(mgr)
	},
	cronjob.CronJobKind: func(mgr manager.Manager) error {
		return cronjob.NewCronJobReconcile(mgr).SetupWithManager(mgr)
	},
	cloneset.CloneSetKind: func(mgr manager.Manager) error {
		return cloneset.NewCloneSetReconciler(mgr).SetupWithManager(mgr)
	},
}

func InitReconcilers(mgr manager.Manager, enables []string) error {
	if err := initClusterMode(mgr); err != nil {
		return err
	}
	log.Info("enable controllers", "name", enables)
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

func EdgeController(mgr manager.Manager) error {
	for {
		if edgecluster.State == string(cluster.Approve) {
			break
		} else {
			log.Info("wait for cluster approved to start remote controller")
			time.Sleep(10 * time.Second)
		}
	}

	// current cluster clientset
	kubeClient, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		log.Errorf("error building kubernetes clientset: %v", err)
	}
	// current cluster dynamic clientset
	dynamicClient, err := dynamic.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		log.Errorf("error building kubernetes dynamic clientset: %v", err)
	}

	kruiseClient, err := kclientset.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		log.Errorf("error building open-kruise clientset: %v", err)
	}

	edge := application.NewReconciler(kubeClient, dynamicClient, kruiseClient)
	go edge.Run(context.Background())
	return nil
}

func initClusterMode(mgr manager.Manager) error {
	next := make(chan struct{})
	go communication.InitCommunication(next)
	<-next
	mode := config.Read().Agent.Mode
	log.Infow("start cluster controller", "mode", mode)
	if mode == "edge" {
		edgecluster.RegisterInit(config.Read().Agent)
		return edgecluster.NewReconciler(mgr).SetupWithManager(mgr)
	}
	if mode == "hub" {
		hub := hubcluster.NewReconciler()
		go hub.Run(context.Background())
		return nil
	}
	return fmt.Errorf("mode %s is not supported yet", mode)
}
