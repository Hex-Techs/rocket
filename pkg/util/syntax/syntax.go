package syntax

import (
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

// [fromParam(PARAMETER)]
// [fromVariable(VARNAME)]

const (
	variablesSyntax = `^\[(from)(P|V)([a-zA-Z]+)\(([a-zA-Z0-9_-]+)\)\]$`
	nameSyntax      = `\([a-zA-Z0-9_-]+\)`
)

type syntaxEngine struct {
	vs *regexp.Regexp
	ns *regexp.Regexp
}

var SyntaxEngine *syntaxEngine

func init() {
	re, err := regexp.Compile(variablesSyntax)
	if err != nil {
		klog.Fatalf("init syntax with error: %v", err)
	}
	ns, err := regexp.Compile(nameSyntax)
	if err != nil {
		klog.Fatalf("init syntax with error: %v", err)
	}
	SyntaxEngine = &syntaxEngine{
		vs: re,
		ns: ns,
	}
}

// ValidateSyntax validate the syntax
func (se *syntaxEngine) ValidateSyntax(s string) bool {
	return se.vs.MatchString(s)
}

// Getvar get the var
func (se *syntaxEngine) GetVar(s string) string {
	tmp := string(se.ns.Find([]byte(s)))
	return strings.TrimRight(strings.TrimLeft(tmp, "("), ")")
}
