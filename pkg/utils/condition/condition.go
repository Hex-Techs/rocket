package condition

import (
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EdgeTraitDeleted = "EdgeTraitDeleted"
)

// generate a new condition
func GenerateCondition(t, reason, msg string, status metav1.ConditionStatus) metav1.Condition {
	return metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               t,
		Status:             status,
		Reason:             reason,
		Message:            msg,
	}
}

// 判断两个 condition 数组内的数据是否相等，计算时剔除了 lastTransitionTime 字段的干扰
func ConditionNotEqual(old, new []metav1.Condition) bool {
	for idx, val := range old {
		val.LastTransitionTime = metav1.NewTime(time.Time{})
		old[idx] = val
	}
	for idx, val := range new {
		val.LastTransitionTime = metav1.NewTime(time.Time{})
		new[idx] = val
	}
	return !cmp.Equal(old, new)
}
