package controllerrevision

import (
	"encoding/json"

	"github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type AuthInfo struct {
	APIServer string `json:"apiServer,omitempty"`
	CAData    []byte `json:"caData,omitempty"`
	CertData  []byte `json:"certData,omitempty"`
	KeyData   []byte `json:"keyData,omitempty"`
	Token     []byte `json:"token,omitempty"`
}

// AI is AuthInfo
func GenerateAI(cluster *v1alpha1.Cluster) *AuthInfo {
	return &AuthInfo{
		APIServer: cluster.Spec.APIServer,
		CAData:    cluster.Spec.CAData,
		CertData:  cluster.Spec.CertData,
		KeyData:   cluster.Spec.KeyData,
		Token:     cluster.Spec.Token,
	}
}

// GenerateCR generates a ControllerRevision for a Cluster
func GenerateCR(cluster *v1alpha1.Cluster) *appsv1.ControllerRevision {
	ai := GenerateAI(cluster)
	b, _ := json.Marshal(ai)
	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: constant.RocketNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "rocket.hextech.io/v1alpha1",
					Kind:       "Cluster",
					UID:        cluster.UID,
					Name:       cluster.Name,
					Controller: pointer.Bool(true),
				},
			},
		},
		Data: runtime.RawExtension{
			Raw: b,
		},
	}
}
