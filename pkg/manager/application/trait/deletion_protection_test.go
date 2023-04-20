package trait

import (
	"encoding/json"
	"reflect"
	"testing"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
)

var (
	deletionProtectionAlways = DeletionProtection{
		Type: "Always",
	}
	deletionProtectionCascading = DeletionProtection{
		Type: "Cascading",
	}
	deletionProtectionOther = DeletionProtection{
		Type: "Other",
	}
)

func Test_deletionProtection_Handler(t *testing.T) {
	always, _ := json.Marshal(deletionProtectionAlways)
	cascading, _ := json.Marshal(deletionProtectionCascading)
	other, _ := json.Marshal(deletionProtectionOther)
	wantalways, wantcascading := constant.Testworkload, constant.Testworkload
	wantalways.Labels = make(map[string]string)
	wantcascading.Labels = make(map[string]string)
	wantalways.Labels[DeleteProtectionLabel] = "Always"
	wantcascading.Labels[DeleteProtectionLabel] = "Cascading"
	setalways, setcascading := constant.Testworkload, constant.Testworkload
	type args struct {
		ttemp    *rocketv1alpha1.Trait
		workload *rocketv1alpha1.Workload
	}
	tests := []struct {
		name    string
		args    args
		want    *rocketv1alpha1.Workload
		wantErr bool
	}{
		{"nulltemplate", args{&rocketv1alpha1.Trait{Kind: DeletionProtectionKind, Template: "abcbsfsdf"}, &constant.Testworkload}, nil, true},
		{"nullworkload", args{&rocketv1alpha1.Trait{Kind: DeletionProtectionKind, Template: "abc"}, nil}, nil, true},
		{"always", args{&rocketv1alpha1.Trait{Kind: DeletionProtectionKind, Template: string(always)}, &setalways}, &wantalways, false},
		{"cascading", args{&rocketv1alpha1.Trait{Kind: DeletionProtectionKind, Template: string(cascading)}, &setcascading}, &wantcascading, false},
		{"other", args{&rocketv1alpha1.Trait{Kind: DeletionProtectionKind, Template: string(other)}, &constant.Testworkload}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &deletionProtection{}
			got, err := d.Handler(tt.args.ttemp, tt.args.workload)
			if (err != nil) != tt.wantErr {
				t.Errorf("deletionProtection.Handler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deletionProtection.Handler() = %v, want %v", got, tt.want)
			}
		})
	}
}
