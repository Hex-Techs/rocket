package validating

import (
	"context"
	"fmt"
	"net/http"

	mapset "github.com/deckarep/golang-set/v2"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ApplicationAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// TODO: reject unsuporrted kind
func (a *ApplicationAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	app := &rocketv1alpha1.Application{}
	err := a.decoder.Decode(req, app)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// a application can not have two same trait
	tset := mapset.NewSet[string]()
	for _, v := range app.Spec.Traits {
		if tset.Contains(v.Kind) {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s only one can be set", v.Kind))
		}
		tset.Add(v.Kind)
	}
	return admission.Allowed("")
}

func (a *ApplicationAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
