package validating

import (
	"context"
	"fmt"
	"net/http"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/syntax"
	"github.com/hex-techs/rocket/pkg/util/tools"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ApplicationAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *ApplicationAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	app := &rocketv1alpha1.Application{}
	err := a.decoder.Decode(req, app)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// variable中不能出现重复的名字
	pSet := tools.New[string]()
	for _, val := range app.Spec.Variables {
		if pSet.Has(val.Name) {
			return admission.Denied(fmt.Sprintf("the variables cannot be repeated '%s'", val.Name))
		}
		pSet.Insert(val.Name)
	}

	// 一种类型的 trait 对于app只能设置一次
	tset := tools.New[string]()
	for _, v := range app.Spec.Traits {
		if tset.Has(v.Kind) {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s only one can be set", v.Kind))
		}
		tset.Insert(v.Kind)
	}

	// 判断template是否可以重复使用
	tname := tools.New[string]()
	for _, temp := range app.Spec.Templates {
		if tname.Has(temp.Name) {
			return admission.Denied(fmt.Sprintf("template '%s' can not reuse in one Application", temp.Name))
		}
		tname.Insert(temp.Name)
		for _, val := range temp.ParameterValues {
			if err := validateStringSyntax(val.Value, pSet); err != nil {
				return admission.Denied(err.Error())
			}
		}
	}
	return admission.Allowed("")
}

func (a *ApplicationAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// Verify whether the slice item in parameters.
func validateStringSyntax(s string, parameters tools.Set[string]) error {
	if syntax.SyntaxEngine.ValidateSyntax(s) {
		name := syntax.SyntaxEngine.GetVar(s)
		if !parameters.Has(name) {
			return fmt.Errorf("cat not found '%s' in variables", name)
		}
	}
	return nil
}
