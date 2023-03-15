package condition

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionNotEqual(t *testing.T) {
	type args struct {
		old []metav1.Condition
		new []metav1.Condition
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"equal", args{old: []metav1.Condition{{Type: "a", Status: "b", LastTransitionTime: metav1.Now()}},
			new: []metav1.Condition{{Type: "a", Status: "b", LastTransitionTime: metav1.NewTime(time.Now().Add(300 * time.Second))}}}, false},
		{"notEqual", args{old: []metav1.Condition{{Type: "a", Status: "b"}},
			new: []metav1.Condition{{Type: "a", Status: "c"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConditionNotEqual(tt.args.old, tt.args.new); got != tt.want {
				t.Errorf("ConditionNotEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
