package cluster

import (
	"context"
	"fmt"

	"github.com/hex-techs/rocket/pkg/utils/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// 集群初始化后，可以被管理的外部组件
const (
	// openKruise
	kruiseRepoURL     = "https://openkruise.github.io/charts/"
	kruiseRepoName    = "openkruise"
	kruiseReleaseName = "kruise"
	kruiseNamespace   = "kruise-system"
	kruiseRepoVersion = "v1.4.0"

	// keda
	kedaRepoURL     = "https://kedacore.github.io/charts"
	kedaRepoName    = "kedacore"
	kedaReleaseName = "keda"
	kedaNamespace   = "keda"
	kedaRepoVersion = "v2.7.2"
)

// 在集群中安装openKruise, keda等模块
func installModulesInCluster(p *config.ModuleParams, mgr manager.Manager, name string) error {
	create, err := configmap(p, mgr)
	if err != nil {
		return err
	}
	if err := buildHelm(rest.CopyConfig(mgr.GetConfig())); err != nil {
		return err
	}
	initRepo()
	if create.kruise {
		if err := installOpenKruise(p.Kruise); err != nil {
			return err
		}
	}
	if create.keda {
		if err := installKeda(p.Keda); err != nil {
			return err
		}
	}
	return nil
}

func initRepo() {
	helmInstance.addOrUpdateChartRepo(&config.HelmRepo{Name: kruiseRepoName, URL: kruiseRepoURL})
	helmInstance.addOrUpdateChartRepo(&config.HelmRepo{Name: kedaRepoName, URL: kedaRepoURL})
	helmInstance.updateChartRepos()
}

func installOpenKruise(param *config.OpenKruise) error {
	if param.Enabled {
		klog.V(3).Infof("install openKruise in namespace '%s' by name '%s'", kruiseNamespace, kruiseReleaseName)
		chartName := fmt.Sprintf("%s/kruise", kruiseRepoName)
		values := fmt.Sprintf(kruiseValues, param.FeatureGates, param.Repository, param.Tag,
			param.LimitCPU, param.LimitMem, param.RequestCPU, param.RequestMem)
		_, err := helmInstance.installOrUpgradeChart(context.TODO(), kruiseReleaseName, chartName,
			kruiseNamespace, kruiseRepoVersion, values)
		return err
	}
	klog.V(3).Info("openKruise is disabled, skip it")
	return nil
}

func installKeda(param *config.Keda) error {
	if param.Enabled {
		klog.V(3).Infof("install keda in namespace '%s' by name '%s'", kedaNamespace, kedaReleaseName)
		if err := handleNamespace(kedaNamespace); err != nil {
			return err
		}
		chartName := fmt.Sprintf("%s/keda", kedaRepoName)
		values := fmt.Sprintf(kedaValues, param.Repository, param.Tag, param.MetricRepository, param.MetricTag)
		_, err := helmInstance.installOrUpgradeChart(context.TODO(), kedaReleaseName, chartName,
			kedaNamespace, kedaRepoVersion, values)
		return err
	}
	klog.V(3).Info("keda is disabled, skip it")
	return nil
}

// 判断namespace是否存在，不存在则创建
func handleNamespace(namespace string) error {
	_, err := helmInstance.kube.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = helmInstance.kube.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
		}
		return err
	}
	return nil
}
