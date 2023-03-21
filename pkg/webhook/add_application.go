package webhook

import (
	"github.com/hex-techs/rocket/pkg/webhook/application/mutating"
	"github.com/hex-techs/rocket/pkg/webhook/application/validating"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

//+kubebuilder:webhook:path=/validate-rocket-hextech-io-v1alpha1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=rocket.hextech.io,resources=application,verbs=create;update,versions=v1alpha1,name=vapplication.kb.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/mutate-rocket-hextech-io-v1alpha1-application,mutating=true,failurePolicy=fail,sideEffects=None,groups=rocket.hextech.io,resources=application,verbs=create;update,versions=v1alpha1,name=mapplication.kb.io,admissionReviewVersions={v1,v1beta1}

// AddToManager adds a webhook to the manager
func serveApplication(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/validate-rocket-hextech-io-v1alpha1-application",
		&webhook.Admission{Handler: &validating.ApplicationAnnotator{Client: mgr.GetClient()}})
	mgr.GetWebhookServer().Register("/mutate-rocket-hextech-io-v1alpha1-application",
		&webhook.Admission{Handler: &mutating.ApplicationAnnotator{}})
}
