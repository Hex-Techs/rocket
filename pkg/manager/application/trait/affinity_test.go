package trait

import (
	"encoding/json"
	"reflect"
	"testing"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	v1 "k8s.io/api/core/v1"
)

var affinityTemp = Affinity{
	NodeAffinity: &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"node1", "node2"},
						},
					},
				},
			},
		},
	},
}

var affiByte, _ = json.Marshal(affinityTemp)

var traitTemp = rocketv1alpha1.Trait{
	Kind:     AffinityKind,
	Template: string(affiByte),
}

func Test_affinity_Handler(t *testing.T) {
	affinityworkloadwithdeploy := constant.Testworkload
	tmpdeploy := constant.DeployTemp
	b, _ := json.Marshal(tmpdeploy)
	affinityworkloadwithdeploy.Spec.Template.Raw = b
	wantaffinityworkloadwithdeploy := constant.Testworkload
	tmpdeploy.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpdeploy)
	wantaffinityworkloadwithdeploy.Spec.Template.Raw = b

	affinityworkloadwithclone := constant.Testworkload
	tmpclone := constant.CloneTemp
	b, _ = json.Marshal(tmpclone)
	affinityworkloadwithclone.Spec.Template.Raw = b
	wantaffinityworkloadwithclone := constant.Testworkload
	tmpclone.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpclone)
	wantaffinityworkloadwithclone.Spec.Template.Raw = b

	affinityworkloadwithsts := constant.Testworkload
	tmpsts := constant.StsTemp
	b, _ = json.Marshal(tmpsts)
	affinityworkloadwithsts.Spec.Template.Raw = b
	wantaffinityworkloadwithsts := constant.Testworkload
	tmpdeploy.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpsts)
	wantaffinityworkloadwithsts.Spec.Template.Raw = b

	affinityworkloadwithests := constant.Testworkload
	tmpests := constant.EstsTemp
	b, _ = json.Marshal(tmpests)
	affinityworkloadwithests.Spec.Template.Raw = b
	wantaffinityworkloadwithests := constant.Testworkload
	tmpdeploy.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpests)
	wantaffinityworkloadwithests.Spec.Template.Raw = b

	affinityworkloadwithcj := constant.Testworkload
	tmpcj := constant.CronjobTemp
	b, _ = json.Marshal(tmpcj)
	affinityworkloadwithcj.Spec.Template.Raw = b
	wantaffinityworkloadwithcj := constant.Testworkload
	tmpcj.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpcj)
	wantaffinityworkloadwithcj.Spec.Template.Raw = b

	affinityworkloadwithj := constant.Testworkload
	tmpj := constant.JobTemp
	b, _ = json.Marshal(tmpj)
	affinityworkloadwithj.Spec.Template.Raw = b
	wantaffinityworkloadwithj := constant.Testworkload
	tmpj.Spec.Template.Spec.Affinity.NodeAffinity = affinityTemp.NodeAffinity
	b, _ = json.Marshal(tmpj)
	wantaffinityworkloadwithj.Spec.Template.Raw = b

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
		{"nulltemplate", args{&rocketv1alpha1.Trait{Kind: AffinityKind, Template: "abcdb"}, &affinityworkloadwithdeploy}, nil, true},
		{"nullworkload", args{&traitTemp, nil}, nil, true},
		{"testdeploy", args{&traitTemp, &affinityworkloadwithdeploy}, &wantaffinityworkloadwithdeploy, false},
		{"testclone", args{&traitTemp, &affinityworkloadwithclone}, &wantaffinityworkloadwithclone, false},
		{"teststs", args{&traitTemp, &affinityworkloadwithsts}, &wantaffinityworkloadwithsts, false},
		{"testests", args{&traitTemp, &affinityworkloadwithests}, &wantaffinityworkloadwithests, false},
		{"testcj", args{&traitTemp, &affinityworkloadwithcj}, &wantaffinityworkloadwithcj, false},
		{"testj", args{&traitTemp, &affinityworkloadwithj}, &wantaffinityworkloadwithj, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &affinity{}
			got, err := a.Handler(tt.args.ttemp, tt.args.workload)
			if (err != nil) != tt.wantErr {
				t.Errorf("affinity.Handler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				g := string(got.Spec.Template.Raw)
				tw := string(tt.want.Spec.Template.Raw)
				if !reflect.DeepEqual(g, tw) {
					t.Errorf("affinity.Handler() = %v\nwant %v", g, tw)
				}
			}
		})
	}
}
