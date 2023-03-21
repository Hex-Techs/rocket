package scheduler

import (
	"context"
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/scheduler/schedule"
	"github.com/hex-techs/rocket/pkg/util/constant"
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
	if workload.Spec.Template.CloneSetTemplate != nil {
		if workload.Spec.Template.CloneSetTemplate.Replicas != nil {
			replicas = int(*workload.Spec.Template.CloneSetTemplate.Replicas)
		}
	}
	if workload.Spec.Template.StatefulSetTemlate != nil {
		if workload.Spec.Template.StatefulSetTemlate.Replicas != nil {
			replicas = int(*workload.Spec.Template.StatefulSetTemlate.Replicas)
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
	ct := workload.Spec.Template.CloneSetTemplate
	cjt := workload.Spec.Template.CronJobTemplate
	stt := workload.Spec.Template.StatefulSetTemlate
	if ct != nil {
		pod.Spec.Containers = ct.Template.Spec.Containers
		if ct.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = ct.Template.Spec.Tolerations
		}
		if ct.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = ct.Template.Spec.Affinity
		}
		if ct.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = ct.Template.Spec.InitContainers
		}
	}
	if cjt != nil {
		pod.Spec.Containers = cjt.JobTemplate.Spec.Template.Spec.Containers
		if cjt.JobTemplate.Spec.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = cjt.JobTemplate.Spec.Template.Spec.Tolerations
		}
		if cjt.JobTemplate.Spec.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = cjt.JobTemplate.Spec.Template.Spec.Affinity
		}
		if cjt.JobTemplate.Spec.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = cjt.JobTemplate.Spec.Template.Spec.InitContainers
		}
	}
	if stt != nil {
		pod.Spec.Containers = stt.Template.Spec.Containers
		if stt.Template.Spec.Tolerations != nil {
			pod.Spec.Tolerations = stt.Template.Spec.Tolerations
		}
		if stt.Template.Spec.Affinity != nil {
			pod.Spec.Affinity = stt.Template.Spec.Affinity
		}
		if stt.Template.Spec.InitContainers != nil {
			pod.Spec.InitContainers = stt.Template.Spec.InitContainers
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
