package validating

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/syntax"
	"github.com/hex-techs/rocket/pkg/util/tools"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type TemplateAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *TemplateAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	template := &rocketv1alpha1.Template{}
	err := a.decoder.Decode(req, template)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	pSet := tools.New[string]()
	for _, val := range template.Spec.Parameters {
		// parameter不能重复
		if pSet.Has(val.Name) {
			return admission.Denied(fmt.Sprintf("the parameter cannot be repeated '%s'", val.Name))
		}
		pSet.Insert(val.Name)
		if err := validateParmeterLegal(val); err != nil {
			return admission.Denied(err.Error())
		}
	}
	if err := validateHostAlias(template.Spec.HostAliases, pSet); err != nil {
		return admission.Denied(err.Error())
	}
	for _, container := range template.Spec.Containers {
		if err := validateEnv(container.Env, pSet); err != nil {
			return admission.Denied(fmt.Sprintf("container '%s': %v", template.Name, err))
		}
		if err := validateCommandAndArgs(append(container.Command, container.Args...), pSet); err != nil {
			return admission.Denied(fmt.Sprintf("container '%s': %v", template.Name, err))
		}
		if err := validateResource(container.Resources, pSet); err != nil {
			return admission.Denied(fmt.Sprintf("container '%s': %v", template.Name, err))
		}
		if err := validateLifecycle(container.Lifecycle, pSet); err != nil {
			return admission.Denied(fmt.Sprintf("container '%s': %v", template.Name, err))
		}
	}
	return admission.Allowed("")
}

func (a *TemplateAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// Verify whether the 'type' and 'default' in the parameter are legal.
func validateParmeterLegal(parameter rocketv1alpha1.Parameter) error {
	if parameter.Default == "" {
		return nil
	}
	if parameter.Type == "number" {
		_, err := strconv.ParseFloat(parameter.Default, 64)
		if err != nil {
			return fmt.Errorf(
				"default value does not match type, type is number, '%s' can not parse to number", parameter.Default)
		}
	}
	if parameter.Type == "boolean" {
		if parameter.Default != "true" && parameter.Default != "false" {
			return fmt.Errorf("default value does not match type, type boolean must be 'true' or 'false'")
		}
	}
	return nil
}

// Verify whether the 'command' and 'args' in the parameter are legal.
func validateCommandAndArgs(c []string, parameters tools.Set[string]) error {
	return validateStringSyntax(c, parameters)
}

// Verify whether 'fromParemeter' in 'env' is legal.
func validateEnv(envs []rocketv1alpha1.Env, parameters tools.Set[string]) error {
	for idx, val := range envs {
		if val.Value == "" && val.FromParam == "" {
			return fmt.Errorf("%s environment must be set 'value' or 'fromParam'", val.Name)
		}
		if val.Value != "" && val.FromParam != "" {
			return fmt.Errorf("%s environment must be set 'value' or 'fromParam', not both", val.Name)
		}
		if val.FromParam == "" {
			continue
		}
		if parameters.Has(syntax.SyntaxEngine.GetVar(val.FromParam)) {
			return nil
		}
		if idx == len(envs)-1 {
			return fmt.Errorf("env '%s' fromParem '%s' not found in paremeters", val.Name, val.FromParam)
		}
	}
	return nil
}

// Verify whether the resources is legal.
func validateResource(resources *rocketv1alpha1.ContainerResource, parameters tools.Set[string]) error {
	if resources == nil {
		return fmt.Errorf("must set resources for template")
	}
	if resources.CPU.Requests == "" ||
		resources.Memory.Requests == "" ||
		resources.EphemeralStorage.Requests == "" {
		return fmt.Errorf("must set requests resource for template")
	}
	// limit 也要验证
	if resources.CPU.Limits == "" ||
		resources.Memory.Limits == "" ||
		resources.EphemeralStorage.Limits == "" {
		return fmt.Errorf("must set limits resource for template")
	}
	r := []string{resources.CPU.Requests, resources.CPU.Limits,
		resources.Memory.Limits, resources.Memory.Requests, resources.EphemeralStorage.Requests,
		resources.EphemeralStorage.Limits}
	return validateStringSyntax(r, parameters)
}

// Verify whether the 'lifecycle' is legal.
func validateLifecycle(lifecycle *v1.Lifecycle, parameters tools.Set[string]) error {
	if lifecycle == nil {
		return nil
	}
	// 只校验了 exec
	if lifecycle.PreStop != nil {
		if lifecycle.PreStop.Exec != nil {
			if err := validateStringSyntax(lifecycle.PreStop.Exec.Command, parameters); err != nil {
				return err
			}
		}
	}
	if lifecycle.PostStart != nil {
		if lifecycle.PostStart.Exec != nil {
			if err := validateStringSyntax(lifecycle.PostStart.Exec.Command, parameters); err != nil {
				return err
			}
		}
	}
	return nil
}

// Verify the HostAlias's IP is legal
func validateHostAlias(h []v1.HostAlias, parameters tools.Set[string]) error {
	ips := []string{}
	for _, v := range h {
		ips = append(ips, v.IP)
	}
	return validateStringSyntax(ips, parameters)
}

// Verify whether the slice item in parameters.
func validateStringSyntax(slice []string, parameters tools.Set[string]) error {
	for _, s := range slice {
		if syntax.SyntaxEngine.ValidateSyntax(s) {
			name := syntax.SyntaxEngine.GetVar(s)
			if !parameters.Has(name) {
				return fmt.Errorf("cat not found '%s' in parameters", name)
			}
		}
	}
	return nil
}
