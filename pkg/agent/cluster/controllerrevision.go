package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type clusterID struct {
	ID string `json:"id,omitempty"`
}

func handleControllerRevision(id string, mgr manager.Manager) (*appsv1.ControllerRevision, error) {
	cli, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		return nil, err
	}
	cr, err := cli.AppsV1().ControllerRevisions(constant.RocketNamespace).Get(context.TODO(), fmt.Sprintf("%s-agent", config.Pread().Name), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			cr, err = cli.AppsV1().ControllerRevisions(constant.RocketNamespace).Create(context.TODO(), generateCR(id), metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}
			return cr, nil
		}
		return nil, err
	}
	return cr, nil
}

func generateCR(id string) *appsv1.ControllerRevision {
	b, _ := json.Marshal(clusterID{ID: id})
	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agent", config.Pread().Name),
			Namespace: constant.RocketNamespace,
		},
		Data: runtime.RawExtension{
			Raw: b,
		},
	}
}
