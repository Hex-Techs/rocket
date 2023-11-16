package application

import (
	"fmt"

	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GenerateWorkload(app *application.Application) (schema.GroupVersionResource, *unstructured.Unstructured) {
	res, workload := generateRes(app.Template.Type)
	labels := map[string]string{
		constant.AppNameLabel:         app.Name,
		constant.AppIDLabel:           fmt.Sprintf("%d", app.ID),
		constant.ManagedByRocketLabel: "rocket",
		constant.ProjectIDLabel:       fmt.Sprintf("%d", app.ProjectID),
		constant.ProjectLabel:         app.Project,
	}
	workload.Object["metadata"] = map[string]interface{}{
		"name":      app.Name,
		"namespace": app.Namespace,
		"labels":    labels,
		"annotations": map[string]string{
			constant.TraitEdgeAnnotation: "true",
		},
	}
	workload.Object["spec"] = map[string]interface{}{
		"selector": map[string]interface{}{
			"matchLabels": labels,
		},
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": labels,
			},
			"spec": map[string]interface{}{
				"containers": []map[string]interface{}{
					{
						"name":       app.Template.Name,
						"image":      app.Template.Image,
						"command":    app.Template.Command,
						"ports":      app.Template.Ports,
						"args":       app.Template.Args,
						"env":        app.Template.Env,
						"workingDir": app.Template.WorkingDir,
						"resources":  app.Template.Resources,
					},
				},
			},
		},
	}
	return res, workload
}

func generateRes(t application.WorkloadType) (schema.GroupVersionResource, *unstructured.Unstructured) {
	var res schema.GroupVersionResource
	var workload *unstructured.Unstructured
	switch t {
	case application.DeploymentType:
		res = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		workload = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
			},
		}
	case application.CronJobType:
		res = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
		workload = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "batch/v1",
				"kind":       "CronJob",
			},
		}
	case application.CloneSetType:
		res = schema.GroupVersionResource{Group: "apps.kruise.io", Version: "v1alpha1", Resource: "clonesets"}
		workload = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps.kruise.io/v1alpha1",
				"kind":       "CloneSet",
			},
		}
	}
	return res, workload
}
