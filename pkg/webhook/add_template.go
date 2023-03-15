package webhook

import (
	"github.com/hex-techs/rocket/pkg/webhook/template/validating"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

//+kubebuilder:webhook:path=/validate-rocket-hextech-io-v1alpha1-template,mutating=false,failurePolicy=fail,sideEffects=None,groups=rocket.hextech.io,resources=templates,verbs=create;update,versions=v1beta1,name=vtemplate.kb.io,admissionReviewVersions={v1,v1beta1}

// AddToManager adds a webhook to the manager
func serveTemplate(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/validate-rocket-hextech-io-v1alpha1-template",
		&webhook.Admission{Handler: &validating.TemplateAnnotator{Client: mgr.GetClient()}})
}
