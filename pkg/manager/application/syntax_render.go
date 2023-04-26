package application

import (
	"fmt"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	lr "github.com/hex-techs/rocket/pkg/utils/resource"
	"github.com/hex-techs/rocket/pkg/utils/syntax"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// type parameterValue struct {
// 	name  string
// 	value string
// }

// - originalParameters app使用的template种定义好的params列表，通过此生成具体的内容
// - generatedParameters 生成后的params
type parameterHandler struct {
	// raw data of Application
	originalVariables []rocketv1alpha1.Variable
	// A list of parameters containing proprietary syntax.
	originalParameterValues []rocketv1alpha1.ParameterValue
	// A list of parameters containing proprietary syntax.
	originalParameters []rocketv1alpha1.Parameter
}

// 渲染 component 中的 parameters 的 key value 并返回
func (p *parameterHandler) renderParameter() map[string]string {
	pv := map[string]string{}
	pStruct := map[string]string{}
	for _, rpv := range p.renderParameterValue() {
		pStruct[rpv.Name] = rpv.Value
	}
	for _, param := range p.originalParameters {
		if val, ok := pStruct[param.Name]; ok {
			pv[param.Name] = val
			continue
		}
		pv[param.Name] = param.Default
	}
	return pv
}

// 将 Application 中的 parameter 通过 variable 渲染出来
func (p *parameterHandler) renderParameterValue() []rocketv1alpha1.ParameterValue {
	pv := []rocketv1alpha1.ParameterValue{}
	for _, acp := range p.originalParameterValues {
		if syntax.SyntaxEngine.ValidateSyntax(acp.Value) {
			name := syntax.SyntaxEngine.GetVar(acp.Value)
			for _, vari := range p.originalVariables {
				if vari.Name == name {
					acp.Value = vari.Value
				}
			}
		}
		pv = append(pv, acp)
	}
	return pv
}

// 渲染 environment
func envRender(envs []rocketv1alpha1.Env, parameters map[string]string) []rocketv1alpha1.Env {
	result := []rocketv1alpha1.Env{}
	for _, env := range envs {
		e := rocketv1alpha1.Env{
			Name: env.Name,
		}
		if env.FromParam != "" {
			e.Value = parameters[syntax.SyntaxEngine.GetVar(env.FromParam)]
		} else {
			e.Value = env.Value
		}
		result = append(result, e)
	}
	return result
}

// 渲染 command 和 arg
func sliceRender(strs []string, parameters map[string]string) []string {
	result := []string{}
	for _, str := range strs {
		val := syntax.SyntaxEngine.GetVar(str)
		if val != "" {
			result = append(result, parameters[val])
		} else {
			result = append(result, str)
		}
	}
	return result
}

func resourceRender(str string, parameters map[string]string) (*resource.Quantity, error) {
	v := stringRender(str, parameters)
	if lr.ResourceEngine.ValidateSyntax(v) {
		i, err := lr.ResourceEngine.GetVar(v)
		if err != nil {
			return nil, err
		}
		quantity, err := resource.ParseQuantity(i)
		if err != nil {
			return nil, err
		}
		return &quantity, nil
	}
	return nil, fmt.Errorf("%s can not parse to resource", v)
}

func lifecycleRender(l *v1.Lifecycle, parameters map[string]string) *v1.Lifecycle {
	if l != nil {
		if l.PostStart != nil {
			if l.PostStart.Exec != nil {
				l.PostStart.Exec.Command = sliceRender(l.PostStart.Exec.Command, parameters)
			}
		}
		if l.PreStop != nil {
			if l.PreStop.Exec != nil {
				l.PreStop.Exec.Command = sliceRender(l.PreStop.Exec.Command, parameters)
			}
		}
	}
	return l
}

func probeRender(p *v1.Probe, parameters map[string]string) *v1.Probe {
	if p == nil {
		return nil
	}
	if p.Exec != nil {
		// probe的定义只在command中可以使用变量
		p.Exec.Command = sliceRender(p.Exec.Command, parameters)
	}
	return p
}

// 渲染 string 格式的内容，主要针对 resources 和 hostalias
func stringRender(str string, parameters map[string]string) string {
	val := syntax.SyntaxEngine.GetVar(str)
	if val != "" {
		return parameters[val]
	}
	return str
}

// 将 string 类型的参数转换为 int32 类型
// func numberRender(str string, parameters map[string]string, dft int32) int32 {
// val := syntax.SyntaxEngine.GetVar(str)
// if val != "" {
// v := parameters[val]
// i, err := strconv.Atoi(v)
// if err != nil {
// klog.V(4).Infof("parse %s to int failed, use default value %d", v, dft)
// return dft
// }
// return int32(i)
// }
// return dft
// }
