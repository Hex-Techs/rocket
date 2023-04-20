package trait

import (
	"encoding/json"
	"reflect"
	"testing"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	v1 "k8s.io/api/core/v1"
)

var nodeAffinity = v1.NodeAffinity{
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
}

var nodeAffiByte, _ = json.Marshal(nodeAffinity)

var nodeAffiJson = string(nodeAffiByte)

var traitTemp = rocketv1alpha1.Trait{
	Kind:     AffinityKind,
	Template: nodeAffiJson,
}

func Test_affinity_Handler(t *testing.T) {
	affinityworkloadwithdeploy := constant.Testworkload
	affinityworkloadwithdeploy.Spec.Template.DeploymentTemplate = &constant.DeployTemp
	wantaffinityworkloadwithdeploy := affinityworkloadwithdeploy
	wantaffinityworkloadwithdeploy.Spec.Template.DeploymentTemplate.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

	affinityworkloadwithclone := constant.Testworkload
	affinityworkloadwithclone.Spec.Template.CloneSetTemplate = &constant.CloneTemp
	wantaffinityworkloadwithclone := affinityworkloadwithclone
	wantaffinityworkloadwithclone.Spec.Template.CloneSetTemplate.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

	affinityworkloadwithsts := constant.Testworkload
	affinityworkloadwithsts.Spec.Template.StatefulSetTemlate = &constant.StsTemp
	wantaffinityworkloadwithsts := affinityworkloadwithsts
	wantaffinityworkloadwithsts.Spec.Template.StatefulSetTemlate.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

	affinityworkloadwithests := constant.Testworkload
	affinityworkloadwithests.Spec.Template.ExtendStatefulSetTemlate = &constant.EstsTemp
	wantaffinityworkloadwithests := affinityworkloadwithests
	wantaffinityworkloadwithests.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

	affinityworkloadwithcronjob := constant.Testworkload
	affinityworkloadwithcronjob.Spec.Template.CronJobTemplate = &constant.CronjobTemp
	wantaffinityworkloadwithcronjob := affinityworkloadwithcronjob
	wantaffinityworkloadwithcronjob.Spec.Template.CronJobTemplate.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

	affinityworkloadwithjob := constant.Testworkload
	affinityworkloadwithjob.Spec.Template.JobTemplate = &constant.JobTemp
	wantaffinityworkloadwithjob := affinityworkloadwithjob
	wantaffinityworkloadwithjob.Spec.Template.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity = &nodeAffinity

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
		{"testcj", args{&traitTemp, &affinityworkloadwithcronjob}, &wantaffinityworkloadwithcronjob, false},
		{"testj", args{&traitTemp, &affinityworkloadwithjob}, &wantaffinityworkloadwithjob, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &affinity{}
			got, err := a.Handler(tt.args.ttemp, tt.args.workload)
			if (err != nil) != tt.wantErr {
				t.Errorf("affinity.Handler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("affinity.Handler() = %v, want %v", got, tt.want)
			}
		})
	}
}
