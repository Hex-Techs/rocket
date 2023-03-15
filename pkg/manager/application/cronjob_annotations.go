package application

import (
	"strconv"
	"strings"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"
)

const (
	ScheduleAnnotation                   = "rocket.hextech.io/schedule"
	SuspendAnnotation                    = "rocket.hextech.io/suspend"
	SuccessfulJobsHistoryLimitAnnotation = "rocket.hextech.io/successful-jobs-history-limit"
	FailedJobsHistoryLimitAnnotation     = "rocket.hextech.io/failed-jobs-history-limit"
	ConcurrencyPolicyAnnotation          = "rocket.hextech.io/concurrency-policy"
	StartingDeadlineSecondsAnnotation    = "rocket.hextech.io/starting-deadline-seconds"
	BackoffLimitAnnotation               = "rocket.hextech.io/backoff-limit"
	TTLSecondsAfterFinishedAnnotation    = "rocket.hextech.io/ttl-seconds-after-finished"
	RestartPolicyAnnotation              = "rocket.hextech.io/restart-policy"
)

var restartPolicy = []string{"Always", "OnFailure", "Never"}

type annotation struct {
	cronjobAnnotation map[string]string
}

func (a *annotation) getSchedule(cmp *rocketv1alpha1.Template) string {
	if s, ok := cmp.GetAnnotations()[ScheduleAnnotation]; ok {
		return s
	} else {
		if s, ok := a.cronjobAnnotation[ScheduleAnnotation]; ok {
			return s
		}
	}
	return ""
}

// 是否将任务挂起，默认不设置，不挂起
func (a *annotation) getSuspend(cmp *rocketv1alpha1.Template) *bool {
	b := false
	if s, ok := a.cronjobAnnotation[SuspendAnnotation]; ok {
		b = strings.ToLower(s) == "true"
	} else {
		if s, ok := cmp.GetAnnotations()[SuspendAnnotation]; ok {
			b = strings.ToLower(s) == "true"
		}
	}
	return &b
}

// 默认一个成功记录
func (a *annotation) getSuccessfulJobsHistoryLimit(cmp *rocketv1alpha1.Template) *int32 {
	l := int32(1)
	if i, ok := a.cronjobAnnotation[SuccessfulJobsHistoryLimitAnnotation]; ok {
		c, err := strconv.ParseInt(i, 10, 32)
		if err != nil {
			klog.V(3).Infof("get successful job history limit with error: %v", err)
			return &l
		}
		l = int32(c)
	} else {
		if i, ok := cmp.GetAnnotations()[SuccessfulJobsHistoryLimitAnnotation]; ok {
			c, err := strconv.ParseInt(i, 10, 32)
			if err != nil {
				klog.V(3).Infof("get successful job history limit by template with error: %v", err)
				return &l
			}
			l = int32(c)
		}
	}
	return &l
}

// 默认一个失败记录
func (a *annotation) getFailedJobsHistoryLimit(cmp *rocketv1alpha1.Template) *int32 {
	l := int32(1)
	if i, ok := a.cronjobAnnotation[FailedJobsHistoryLimitAnnotation]; ok {
		c, err := strconv.ParseInt(i, 10, 32)
		if err != nil {
			klog.V(3).Infof("get failed job history limit with error: %v", err)
			return &l
		}
		l = int32(c)
	} else {
		if i, ok := cmp.GetAnnotations()[FailedJobsHistoryLimitAnnotation]; ok {
			c, err := strconv.ParseInt(i, 10, 32)
			if err != nil {
				klog.V(3).Infof("get failed job history limit by template with error: %v", err)
				return &l
			}
			l = int32(c)
		}
	}
	return &l
}

// 默认不允许同时存在
func (a *annotation) getConcurrencyPolicy(cmp *rocketv1alpha1.Template) batchv1.ConcurrencyPolicy {
	b := batchv1.ForbidConcurrent
	if c, ok := a.cronjobAnnotation[ConcurrencyPolicyAnnotation]; ok {
		con := strings.ToLower(c)
		switch con {
		case "allow":
			b = batchv1.AllowConcurrent
		case "forbid":
			b = batchv1.ForbidConcurrent
		case "replace":
			b = batchv1.ReplaceConcurrent
		}
	} else {
		if c, ok := cmp.GetAnnotations()[ConcurrencyPolicyAnnotation]; ok {
			con := strings.ToLower(c)
			switch con {
			case "allow":
				b = batchv1.AllowConcurrent
			case "forbid":
				b = batchv1.ForbidConcurrent
			case "replace":
				b = batchv1.ReplaceConcurrent
			}
		}
	}
	return b
}

func (a *annotation) getStartingDeadlineSeconds(cmp *rocketv1alpha1.Template) *int64 {
	l := int64(30)
	if i, ok := a.cronjobAnnotation[StartingDeadlineSecondsAnnotation]; ok {
		c, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			klog.V(3).Infof("get starting deadline seconds with error: %v", err)
			return &l
		}
		l = c
	} else {
		if i, ok := cmp.GetAnnotations()[StartingDeadlineSecondsAnnotation]; ok {
			c, err := strconv.ParseInt(i, 10, 64)
			if err != nil {
				klog.V(3).Infof("get starting deadline seconds by template with error: %v", err)
				return &l
			}
			l = c
		}
	}
	return &l
}

// 默认不重试
func (a *annotation) getBackoffLimit(cmp *rocketv1alpha1.Template) *int32 {
	l := int32(0)
	if i, ok := a.cronjobAnnotation[BackoffLimitAnnotation]; ok {
		c, err := strconv.ParseInt(i, 10, 32)
		if err != nil {
			klog.V(3).Infof("get failed job history limit with error: %v", err)
			return &l
		}
		l = int32(c)
	} else {
		if i, ok := cmp.GetAnnotations()[BackoffLimitAnnotation]; ok {
			c, err := strconv.ParseInt(i, 10, 32)
			if err != nil {
				klog.V(3).Infof("get failed job history limit by template with error: %v", err)
				return &l
			}
			l = int32(c)
		}
	}
	return &l
}

// 默认 24 小时
func (a *annotation) getTTLSecondsAfterFinished(cmp *rocketv1alpha1.Template) *int32 {
	l := int32(86400)
	if i, ok := a.cronjobAnnotation[TTLSecondsAfterFinishedAnnotation]; ok {
		c, err := strconv.ParseInt(i, 10, 32)
		if err != nil {
			klog.V(3).Infof("get failed job history limit with error: %v", err)
			return &l
		}
		l = int32(c)
	} else {
		if i, ok := cmp.GetAnnotations()[TTLSecondsAfterFinishedAnnotation]; ok {
			c, err := strconv.ParseInt(i, 10, 32)
			if err != nil {
				klog.V(3).Infof("get failed job history limit by template with error: %v", err)
				return &l
			}
			l = int32(c)
		}
	}
	return &l
}

func (a *annotation) getRestartPolicy(cmp *rocketv1alpha1.Template) v1.RestartPolicy {
	if r, ok := a.cronjobAnnotation[RestartPolicyAnnotation]; ok {
		if slices.Contains(restartPolicy, r) {
			return v1.RestartPolicy(r)
		}
	} else {
		if r, ok := cmp.GetAnnotations()[RestartPolicyAnnotation]; ok {
			if slices.Contains(restartPolicy, r) {
				return v1.RestartPolicy(r)
			}
		}
	}
	return v1.RestartPolicyNever
}
