package resouce

import (
	"regexp"
	"testing"
)

func Test_resourceEngine_ValidateSyntax(t *testing.T) {
	type fields struct {
		cr *regexp.Regexp
		mr *regexp.Regexp
	}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"test01", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Ki"}, true},
		{"test02", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Mi"}, true},
		{"test03", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Gi"}, true},
		{"test04", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "100m"}, true},
		{"test05", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "100"}, true},
		{"test06", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "123GGJ"}, false},
		{"test07", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "sdf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &resourceEngine{
				cr: tt.fields.cr,
				mr: tt.fields.mr,
			}
			if got := r.ValidateSyntax(tt.args.s); got != tt.want {
				t.Errorf("resourceEngine.ValidateSyntax() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_resourceEngine_GetVar(t *testing.T) {
	type fields struct {
		cr *regexp.Regexp
		mr *regexp.Regexp
	}
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{"test01", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Ki"}, "10Ki", false},
		{"test02", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Mi"}, "10Mi", false},
		{"test03", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "10Gi"}, "10Gi", false},
		{"test04", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "100m"}, "100m", false},
		{"test04", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "100"}, "100", false},
		{"test05", fields{cr: ResourceEngine.cr, mr: ResourceEngine.mr}, args{s: "sdfk"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &resourceEngine{
				cr: tt.fields.cr,
				mr: tt.fields.mr,
			}
			got, err := r.GetVar(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("resourceEngine.GetVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resourceEngine.GetVar() = %v, want %v", got, tt.want)
			}
		})
	}
}
