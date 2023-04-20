package scheduler

import (
	"context"
	"encoding/json"
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/scheduler/schedule"
	"github.com/hex-techs/rocket/pkg/util/constant"
	"github.com/hex-techs/rocket/pkg/util/gvktools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/klog/v2"
)

const (
	DefaultEnv  = "test"
	DefaultArea = "public"
)

type ClusterResult struct {
	// 集群名称
	Name string `json:"name,omitempty"`
	// 调度分数，分数越高，越适合
	Score int64 `json:"score,omitempty"`
	// 集群的地域信息 eg: ap-beijing
	Region string `json:"region,omitempty"`
	// 集群的区域信息 eg: pub, ded
	Area string `json:"area,omitempty"`
	// 集群的环境信息 eg: tst, pre, pd
	Env string `json:"env,omitempty"`
}

func Scheduler(workload *rocketv1alpha1.Workload) []ClusterResult {
	klog.V(4).Infof("Scheduler workload %#v", workload)
	var scoreMap = []ClusterResult{}
	regions := mapset.NewSet[string]()
	for i := range workload.Spec.Regions {
		regions.Add(workload.Spec.Regions[i])
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromWorkload(workload)
	if err != nil {
		klog.Errorf("get resource and gvk from workload failed, err: %v", err)
		return nil
	}
	if resource == nil {
		klog.Errorf("get resource and gvk from workload failed, err: resource is nil")
		return nil
	}
	gvr := gvktools.SetGVRForWorkload(gvk)
	b, _ := json.Marshal(resource)
	ClientMap.Range(func(key, value interface{}) bool {
		name := key.(string)
		v := value.(*ClusterClient)
		if !clusterTolerateFilter(v.Taint, workload.Spec.Tolerations) {
			return true
		}
		area, env := areaEnv(workload.Labels)
		if !regions.Contains(v.Region) || v.Area != area || v.Env != env {
			return true // continue
		}
		scoreMap = append(scoreMap, ClusterResult{
			Name:   name,
			Score:  0,
			Region: v.Region,
			Area:   string(v.Area),
			Env:    string(v.Env),
		})
		for _, node := range v.Controller().List() {
			p := newPod(workload)
			if schedule.Filter(context.TODO(), p, node) {
				// 为每一个符合要求的集群和节点计算分数
				for idx, cluster := range scoreMap {
					if cluster.Name == name {
						scoreMap[idx].Score = scoreMap[idx].Score + schedule.Score(context.TODO(), p, node)
					}
				}
			}
		}
		return true
	})
	replicas := 1
	switch gvr.Resource {
	case "deployments":
		deploy := &appsv1.Deployment{}
		json.Unmarshal(b, deploy)
		if deploy.Spec.Replicas != nil {
			replicas = int(*deploy.Spec.Replicas)
		}
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		if clone.Spec.Replicas != nil {
			replicas = int(*clone.Spec.Replicas)
		}
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			if sts.Spec.Replicas != nil {
				replicas = int(*sts.Spec.Replicas)
			}
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			if sts.Spec.Replicas != nil {
				replicas = int(*sts.Spec.Replicas)
			}
		}
	}
	// 将不符合replicas要求的集群剔除
	for idx, cluster := range scoreMap {
		if cluster.Score < int64(replicas) {
			if idx == len(scoreMap) {
				scoreMap = scoreMap[:idx]
			} else {
				scoreMap = append(scoreMap[:idx], scoreMap[idx+1:]...)
			}
		}
	}
	s := sortResult{d: scoreMap}
	sort.Sort(s)
	return s.d
}
func newPod(workload *rocketv1alpha1.Workload) *v1.Pod {
	pod := &v1.Pod{
		Spec: v1.PodSpec{},
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromWorkload(workload)
	if err != nil {
		klog.Errorf("get resource and gvk from workload failed, err: %v", err)
		return nil
	}
	if resource == nil {
		klog.Errorf("get resource and gvk from workload failed, err: resource is nil")
		return nil
	}
	gvr := gvktools.SetGVRForWorkload(gvk)
	b, _ := json.Marshal(resource)
	switch gvr.Resource {
	case "deployments":
		deploy := &appsv1.Deployment{}
		json.Unmarshal(b, deploy)
		pod.Spec.Containers = deploy.Spec.Template.Spec.Containers
		if deploy.Spec.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = deploy.Spec.Template.Spec.Tolerations
		}
		if deploy.Spec.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = deploy.Spec.Template.Spec.Affinity
		}
		if deploy.Spec.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = deploy.Spec.Template.Spec.InitContainers
		}
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		pod.Spec.Containers = clone.Spec.Template.Spec.Containers
		if clone.Spec.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = clone.Spec.Template.Spec.Tolerations
		}
		if clone.Spec.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = clone.Spec.Template.Spec.Affinity
		}
		if clone.Spec.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = clone.Spec.Template.Spec.InitContainers
		}
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			pod.Spec.Containers = sts.Spec.Template.Spec.Containers
			if sts.Spec.Template.Spec.Tolerations != nil {
				pod.Spec.Tolerations = sts.Spec.Template.Spec.Tolerations
			}
			if sts.Spec.Template.Spec.Affinity != nil {
				pod.Spec.Affinity = sts.Spec.Template.Spec.Affinity
			}
			if sts.Spec.Template.Spec.InitContainers != nil {
				pod.Spec.InitContainers = sts.Spec.Template.Spec.InitContainers
			}
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			pod.Spec.Containers = sts.Spec.Template.Spec.Containers
			if sts.Spec.Template.Spec.Tolerations != nil {
				pod.Spec.Tolerations = sts.Spec.Template.Spec.Tolerations
			}
			if sts.Spec.Template.Spec.Affinity != nil {
				pod.Spec.Affinity = sts.Spec.Template.Spec.Affinity
			}
			if sts.Spec.Template.Spec.InitContainers != nil {
				pod.Spec.InitContainers = sts.Spec.Template.Spec.InitContainers
			}
		}
	case "cronjobs":
		cronjob := &batchv1.CronJob{}
		json.Unmarshal(b, cronjob)
		pod.Spec.Containers = cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
		if cronjob.Spec.JobTemplate.Spec.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = cronjob.Spec.JobTemplate.Spec.Template.Spec.Tolerations
		}
		if cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity
		}
		if cronjob.Spec.JobTemplate.Spec.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = cronjob.Spec.JobTemplate.Spec.Template.Spec.InitContainers
		}
	case "jobs":
		job := &batchv1.Job{}
		json.Unmarshal(b, job)
		pod.Spec.Containers = job.Spec.Template.Spec.Containers
		if job.Spec.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = job.Spec.Template.Spec.Tolerations
		}
		if job.Spec.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = job.Spec.Template.Spec.Affinity
		}
		if job.Spec.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = job.Spec.Template.Spec.InitContainers
		}
	}
	return pod
}

// 集群污点与容忍
func clusterTolerateFilter(taints []v1.Taint, tolerations []v1.Toleration) bool {
	filterPredicate := func(t *v1.Taint) bool {
		// PodToleratesNodeTaints is only interested in NoSchedule and NoExecute taints.
		return t.Effect == v1.TaintEffectNoSchedule || t.Effect == v1.TaintEffectNoExecute
	}
	taint, isUntolerated := v1helper.FindMatchingUntoleratedTaint(taints, tolerations, filterPredicate)
	if !isUntolerated {
		return true
	}
	klog.V(4).Infof("node(s) had taint {%s: %s}, that the workload didn't tolerate",
		taint.Key, taint.Value)
	return false
}

func areaEnv(labels map[string]string) (area, env string) {
	if a, ok := labels[constant.CloudAreaLabel]; !ok {
		klog.V(3).Infof("unable to obtain area when scheduling workload, use default '%s'", DefaultArea)
		area = DefaultArea
	} else {
		area = a
	}
	if e, ok := labels[constant.EnvLabel]; !ok {
		klog.V(3).Infof("unable to obtain env when scheduling workload, use default '%s'", DefaultEnv)
		env = DefaultEnv
	} else {
		env = e
	}
	return area, env
}

type sortResult struct {
	d []ClusterResult
}

func (s sortResult) Len() int {
	return len(s.d)
}
func (s sortResult) Less(i, j int) bool {
	// 降序排序
	return s.d[i].Score > s.d[j].Score
}
func (s sortResult) Swap(i, j int) {
	s.d[i], s.d[j] = s.d[j], s.d[i]
}
