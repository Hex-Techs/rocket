package webhook

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func InitWebhook(mgr manager.Manager) {
	serveTemplate(mgr)
	serveApplication(mgr)
}
