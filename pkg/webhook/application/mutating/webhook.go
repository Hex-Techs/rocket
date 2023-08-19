package mutating

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ApplicationAnnotator struct {
	// Client  client.Client
	decoder *admission.Decoder
}

func (a *ApplicationAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	app := &rocketv1alpha1.Application{}
	err := a.decoder.Decode(req, app)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	annoSet := app.Annotations
	if len(annoSet) == 0 {
		annoSet = make(map[string]string)
	}
	// 将edge类型的trait kind设置到annotation中
	et := []string{}
	for _, c := range app.Spec.Traits {
		if constant.EdgeTrait.Contains(c.Kind) {
			et = append(et, c.Kind)
		}
	}
	if len(et) != 0 {
		annoSet[constant.TraitEdgeAnnotation] = strings.Join(et, ",")
	}
	app.Annotations = annoSet
	marshaledRF, err := json.Marshal(app)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRF)
}

func (a *ApplicationAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
