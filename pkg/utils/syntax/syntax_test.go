package syntax

import (
	"regexp"
	"testing"
)

func Test_syntaxEngine_ValidateSyntax(t *testing.T) {
	type fields struct {
		vs *regexp.Regexp
		ns *regexp.Regexp
	}
	type args struct {
		s string
	}
	vs, ns := initRe()
	f := fields{
		vs: vs,
		ns: ns,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "test01", fields: f, args: args{s: "[fromParam(test)]"}, want: true},
		{name: "test02", fields: f, args: args{s: "[fromParam(test.)]"}, want: false},
		{name: "test03", fields: f, args: args{s: "[fromParam()]"}, want: false},
		{name: "test04", fields: f, args: args{s: "test"}, want: false},
		{name: "test05", fields: f, args: args{s: "[fromVariable(test)]"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &syntaxEngine{
				vs: tt.fields.vs,
				ns: tt.fields.ns,
			}
			if got := se.ValidateSyntax(tt.args.s); got != tt.want {
				t.Errorf("syntaxEngine.ValidateSyntax() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_syntaxEngine_GetVar(t *testing.T) {
	type fields struct {
		vs *regexp.Regexp
		ns *regexp.Regexp
	}
	type args struct {
		s string
	}
	vs, ns := initRe()
	f := fields{
		vs: vs,
		ns: ns,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{name: "test01", fields: f, args: args{s: "[fromParam(test)]"}, want: "test"},
		{name: "test02", fields: f, args: args{s: "[fromParam(test.)]"}, want: ""},
		{name: "test03", fields: f, args: args{s: "[fromParam()]"}, want: ""},
		{name: "test04", fields: f, args: args{s: "test"}, want: ""},
		{name: "test05", fields: f, args: args{s: "[fromVariable(test)]"}, want: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &syntaxEngine{
				vs: tt.fields.vs,
				ns: tt.fields.ns,
			}
			if got := se.GetVar(tt.args.s); got != tt.want {
				t.Errorf("syntaxEngine.GetVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func initRe() (*regexp.Regexp, *regexp.Regexp) {
	re, _ := regexp.Compile(variablesSyntax)
	ns, _ := regexp.Compile(nameSyntax)
	return re, ns
}
