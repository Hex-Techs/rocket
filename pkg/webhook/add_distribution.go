package webhook

import (
	"github.com/hex-techs/rocket/pkg/webhook/distribution/validating"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

//+kubebuilder:webhook:path=/validate-rocket-hextech-io-v1alpha1-distribution,mutating=false,failurePolicy=fail,sideEffects=None,groups=rocket.hextech.io,resources=distributions,verbs=create;update,versions=v1alpha1,name=vdistribution.kb.io,admissionReviewVersions={v1,v1beta1}

// AddToManager adds a webhook to the manager
func servedistribution(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/validate-rocket-hextech-io-v1alpha1-distribution",
		&webhook.Admission{Handler: &validating.DistributionAnnotator{}})
}
