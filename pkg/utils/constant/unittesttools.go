package constant

import (
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Testapplication is a application for test
var Testapplication = rocketv1alpha1.Application{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: rocketv1alpha1.ApplicationSpec{
		Template: runtime.RawExtension{},
	},
}

// PodTemp is a pod template for test
var PodTemp = v1.PodTemplateSpec{
	Spec: v1.PodSpec{
		Affinity: &v1.Affinity{},
	},
}

// DeployTemp is a deployment template for test
var DeployTemp = appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: appsv1.DeploymentSpec{
		Template: PodTemp,
	},
}

// CloneTemp is a cloneset template for test
var CloneTemp = kruiseappsv1alpha1.CloneSet{
	TypeMeta: metav1.TypeMeta{
		Kind:       "CloneSet",
		APIVersion: "apps.kruise.io/v1alpha1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: kruiseappsv1alpha1.CloneSetSpec{
		Template: PodTemp,
	},
}

// StsTemp is a statefulset template for test
var StsTemp = appsv1.StatefulSet{
	TypeMeta: metav1.TypeMeta{
		Kind:       "StatefulSet",
		APIVersion: "apps/v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: appsv1.StatefulSetSpec{
		Template: PodTemp,
	},
}

// EstsTemp is a extendstatefulset template for test
var EstsTemp = kruiseappsv1beta1.StatefulSet{
	TypeMeta: metav1.TypeMeta{
		Kind:       "StatefulSet",
		APIVersion: "apps.kruise.io/v1beta1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: kruiseappsv1beta1.StatefulSetSpec{
		Template: PodTemp,
	},
}

// CronJobTemp is a cronjob template for test
var CronjobTemp = batchv1.CronJob{
	TypeMeta: metav1.TypeMeta{
		Kind:       "CronJob",
		APIVersion: "batch/v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: batchv1.CronJobSpec{
		JobTemplate: batchv1.JobTemplateSpec{
			Spec: JobTemp.Spec,
		},
	},
}

// JobTemp is a job template for test
var JobTemp = batchv1.Job{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Job",
		APIVersion: "batch/v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	Spec: batchv1.JobSpec{
		Template: PodTemp,
	},
}
