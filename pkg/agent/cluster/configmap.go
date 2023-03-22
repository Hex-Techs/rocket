package cluster

import (
	"context"
	"encoding/json"

	"github.com/google/go-cmp/cmp"
	agentconfig "github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const cmName = "hextech-agent-component"

type createTag struct {
	kruise bool
	keda   bool
}

// bool 判断是否执行 helm
func configmap(p *agentconfig.ModuleParams, mgr manager.Manager) (*createTag, error) {
	ct := &createTag{}
	gcm := generateConfigMap(p)
	ctx := context.TODO()
	cli, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		return nil, err
	}
	cm, err := cli.CoreV1().ConfigMaps(constant.RocketNamespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = cli.CoreV1().ConfigMaps(constant.RocketNamespace).Create(ctx, gcm, metav1.CreateOptions{})
			if err != nil {
				return ct, err
			}
			ct.kruise, ct.keda = true, true
			return ct, nil
		} else {
			return ct, err
		}
	}
	kruise := isCreateOrNot("kruise", gcm.Data, cm.DeepCopy().Data)
	keda := isCreateOrNot("keda", gcm.Data, cm.DeepCopy().Data)
	if kruise || keda {
		_, err = cli.CoreV1().ConfigMaps(constant.RocketNamespace).Update(ctx, gcm, metav1.UpdateOptions{})
		if err != nil {
			return ct, err
		}
	}
	ct.kruise = kruise
	ct.keda = keda
	return ct, nil
}

func isCreateOrNot(key string, data1, data2 map[string]string) bool {
	return !cmp.Equal(data1[key], data2[key])
}

func generateConfigMap(p *agentconfig.ModuleParams) *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: constant.RocketNamespace,
		},
		Data: map[string]string{},
	}
	if p.Kruise != nil {
		if p.Kruise.Enabled {
			b, _ := json.Marshal(p.Kruise)
			cm.Data["kruise"] = string(b)
		}
	}
	if p.Keda != nil {
		if p.Keda.Enabled {
			b, _ := json.Marshal(p.Keda)
			cm.Data["keda"] = string(b)
		}
	}
	return cm
}
