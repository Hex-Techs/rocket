package validating

import (
	"context"
	"fmt"
	"net/http"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// 只支持 Namspace, ConfigMap, Secret, ResourceQuota 四种资源
var kind = map[string]struct{}{
	"Namespace":     {},
	"ConfigMap":     {},
	"Secret":        {},
	"ResourceQuota": {},
}

type DistributionAnnotator struct {
	decoder *admission.Decoder
}

func (r *DistributionAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	rd := &rocketv1alpha1.Distribution{}
	err := r.decoder.Decode(req, rd)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if len(rd.Spec.Resource.Raw) != 0 {
		obj, _, err := unstructured.UnstructuredJSONScheme.Decode(rd.Spec.Resource.Raw, nil, nil)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if _, ok := kind[obj.GetObjectKind().GroupVersionKind().Kind]; !ok {
			return admission.Denied(fmt.Sprintf("the resource kind '%s' is not supported", obj.GetObjectKind().GroupVersionKind().Kind))
		}
	}
	return admission.Allowed("")
}
func (r *DistributionAnnotator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
