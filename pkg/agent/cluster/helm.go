package cluster

import (
	"context"

	agentconfig "github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	// default repo cache
	defaultRepositoryCache = "/tmp/.helmcache"
	// default helm home
	defaultRepositoryConfig = "/tmp/.helmrepo"
)

// 初始化 helm 应用
func buildHelm(kubeconfig *rest.Config) error {
	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			RepositoryCache:  defaultRepositoryCache,
			RepositoryConfig: defaultRepositoryConfig,
			Debug:            true,
			Linting:          true,
			Namespace:        constant.RocketNamespace,
		},
		RestConfig: kubeconfig,
	}
	helmClient, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		return err
	}
	helmInstance = &helmApplication{
		client: helmClient,
		kube:   kubernetes.NewForConfigOrDie(kubeconfig),
	}
	return nil
}

// helm 管理的应用，一般是系统内部应用
type helmApplication struct {
	client helm.Client
	kube   kubernetes.Interface
}

var helmInstance *helmApplication

// 添加或升级 helm chart repo
func (h *helmApplication) addOrUpdateChartRepo(c *agentconfig.HelmRepo) error {
	chartRepo := repo.Entry{
		Name: c.Name,
		URL:  c.URL,
	}
	if c.NeedAuth() {
		chartRepo.Username = c.User
		chartRepo.Password = c.Password
	}
	return h.client.AddOrUpdateChartRepo(chartRepo)
}

func (h *helmApplication) updateChartRepos() error {
	klog.V(1).Info("update helm repository")
	return h.client.UpdateChartRepos()
}

// 安装或升级 helm chart，无需等待直接返回成功或失败
func (h *helmApplication) installOrUpgradeChart(ctx context.Context, name, chart, namespace, version, values string) (
	*release.Release, error) {
	chartSpec := helm.ChartSpec{
		ReleaseName: name,
		ChartName:   chart,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Version:     version,
		ValuesYaml:  values,
	}
	klog.V(3).Infof("install or upgrade chart with chart spec: %#v", chartSpec)
	return h.client.InstallOrUpgradeChart(ctx, &chartSpec, nil)
}
