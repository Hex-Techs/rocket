package constant

import (
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Testworkload is a workload for test
var Testworkload = rocketv1alpha1.Workload{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: rocketv1alpha1.WorkloadSpec{
		Template: rocketv1alpha1.WorkloadTemplate{},
	},
}

// PodTemp is a pod template for test
var PodTemp = v1.PodTemplateSpec{
	Spec: v1.PodSpec{
		Affinity: &v1.Affinity{},
	},
}

// DeployTemp is a deployment template for test
var DeployTemp = appsv1.DeploymentSpec{
	Template: PodTemp,
}

// CloneTemp is a cloneset template for test
var CloneTemp = kruiseappsv1alpha1.CloneSetSpec{
	Template: PodTemp,
}

// StsTemp is a statefulset template for test
var StsTemp = appsv1.StatefulSetSpec{
	Template: PodTemp,
}

// EstsTemp is a extendstatefulset template for test
var EstsTemp = kruiseappsv1beta1.StatefulSetSpec{
	Template: PodTemp,
}

// CronJobTemp is a cronjob template for test
var CronjobTemp = batchv1.CronJobSpec{
	JobTemplate: JobTemp,
}

// JobTemp is a job template for test
var JobTemp = batchv1.JobTemplateSpec{
	Spec: batchv1.JobSpec{
		Template: PodTemp,
	},
}
