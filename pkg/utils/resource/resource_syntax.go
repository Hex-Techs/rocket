package resouce

import (
	"fmt"
	"os"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Gi, Mi, Ki supported 1Gi = 2^10Mi = 2^20Ki
const (
	cpuRegexp = `^([0-9]+)(m|)$`
	memRegexp = `^([0-9]+)(Gi|Mi|Ki)$`
)

type resourceEngine struct {
	cr *regexp.Regexp
	mr *regexp.Regexp
}

var ResourceEngine *resourceEngine

func init() {
	cr, err := regexp.Compile(cpuRegexp)
	if err != nil {
		log.Log.Error(err, "init resource_syntax")
		os.Exit(1)
	}
	mr, err := regexp.Compile(memRegexp)
	if err != nil {
		log.Log.Error(err, "init resource_syntax")
		os.Exit(1)
	}
	ResourceEngine = &resourceEngine{
		cr: cr,
		mr: mr,
	}
}

func (r *resourceEngine) ValidateSyntax(s string) bool {
	if !r.cr.MatchString(s) {
		return r.mr.MatchString(s)
	}
	return true
}

// 获取数据，对于 CPU 只需要返回数据，对于内存则根据单位区分
func (r *resourceEngine) GetVar(s string) (string, error) {
	tmp := string(r.mr.Find([]byte(s)))
	if tmp == "" {
		tmp = string(r.cr.Find([]byte(s)))
	}
	if tmp == "" {
		return "", fmt.Errorf("data error, '%s' can not get data", tmp)
	}
	return tmp, nil
}
