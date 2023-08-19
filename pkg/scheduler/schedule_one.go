package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/scheduler/filter"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
}

func (r *SchedulerReconciler) scheduler(ctx context.Context, application *rocketv1alpha1.Application) []ClusterResult {
	log := log.FromContext(ctx)
	if len(r.schedCache) == 0 {
		log.V(3).Info("no cluster can be scheduled", "application", tools.KObj(application))
	}
	var scoreMap = []ClusterResult{}
	regions := mapset.NewSet[string]()
	for i := range application.Spec.Regions {
		regions.Add(application.Spec.Regions[i])
	}
	area, err := r.areaEnv(application.Name, application.Namespace)
	if err != nil {
		log.Error(err, "unable to scheduler application", "application", tools.KObj(application))
		r.recoder.Eventf(application, v1.EventTypeWarning, "SchedulerFailed",
			"unable to scheduler application '%s/%s' and got error: %v", application.Namespace, application.Name, err)
		return scoreMap
	}
	p, err := newPod(ctx, application)
	if err != nil {
		log.Error(err, "unable to scheduler application", "application", tools.KObj(application))
		r.recoder.Eventf(application, v1.EventTypeWarning, "SchedulerFailed",
			"unable to scheduler application '%s/%s' and got error: %v", application.Namespace, application.Name, err)
		return scoreMap
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	log.V(3).Info("Attempting to schedule application", "application", tools.KObj(application))
	for _, cluster := range r.schedCache {
		if !clusterTolerateFilter(cluster.Taint, application.Spec.Tolerations) {
			continue
		}
		if !regions.Contains(cluster.region) || cluster.area != area {
			continue
		}
		result, err := r.scheddulePod(ctx, cluster.name, p)
		if err != nil {
			log.Error(err, "unable to scheduler application", "application", tools.KObj(application))
			continue
		}
		scoreMap = append(scoreMap, *result)
	}
	log.V(3).Info("scheduler application in selected", "application", tools.KObj(application), "allScore", scoreMap)
	scoreMap = r.removeNotMatchCluster(application, scoreMap)
	s := sortResult{d: scoreMap}
	sort.Sort(s)
	log.V(3).Info("scheduler application done", "application", tools.KObj(application), "resultScore", s.d)
	return s.d
}

func (r *SchedulerReconciler) areaEnv(name, namespace string) (area string, err error) {
	app := rocketv1alpha1.Application{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &app)
	if err != nil {
		return "", err
	}
	if app.Spec.CloudArea == "" {
		return "", fmt.Errorf("%s cloud area is empty", name)
	}
	if app.Spec.Environment == "" {
		return "", fmt.Errorf("%s environment is empty", name)
	}
	return app.Spec.CloudArea, nil
}

func (r *SchedulerReconciler) scheddulePod(ctx context.Context, clustername string, pod *v1.Pod) (result *ClusterResult, err error) {
	log := log.FromContext(ctx)
	if _, ok := r.snapshot[clustername]; !ok {
		return result, fmt.Errorf("cluster %s not found in snapshot, will retry", clustername)
	}
	if err := r.schedCache[clustername].schedCache.UpdateSnapshot(r.snapshot[clustername]); err != nil {
		return result, err
	}
	log.V(5).Info("Snapshotting scheduler cache and node infos done")
	if r.snapshot[clustername].NumNodes() == 0 {
		return result, fmt.Errorf("cluster %s has no available nodes", clustername)
	}
	allNodes, err := r.snapshot[clustername].NodeInfos().List()
	if err != nil {
		return result, err
	}
	result = &ClusterResult{
		Name:   clustername,
		Region: r.schedCache[clustername].region,
		Area:   string(r.schedCache[clustername].area),
	}
	for _, n := range allNodes {
		if filter.Filter(ctx, pod, n) {
			result.Score = result.Score + filter.Score(ctx, pod, n)
		}
	}
	return result, nil
}

func (r *SchedulerReconciler) removeNotMatchCluster(application *rocketv1alpha1.Application, scoreMap []ClusterResult) []ClusterResult {
	desired := 1
	replicas := application.Spec.Replicas
	if replicas != nil {
		desired = int(*replicas)
	}
	result := []ClusterResult{}
	// 将不符合replicas要求的集群剔除
	for _, cluster := range scoreMap {
		if cluster.Score >= int64(desired) {
			result = append(result, cluster)
		}
	}
	return result
}

func newPod(ctx context.Context, application *rocketv1alpha1.Application) (*v1.Pod, error) {
	log := log.FromContext(ctx)
	pod := &v1.Pod{
		Spec: v1.PodSpec{},
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromApplication(application)
	if err != nil {
		return nil, err
	}
	gvr := gvktools.SetGVRForApplication(gvk)
	b, _ := json.Marshal(resource)
	switch gvr.Resource {
	case "deployments":
		deploy := &appsv1.Deployment{}
		json.Unmarshal(b, deploy)
		pod.Spec = deploy.Spec.Template.Spec
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		pod.Spec = clone.Spec.Template.Spec
	case "cronjobs":
		cj := &batchv1.CronJob{}
		json.Unmarshal(b, cj)
		pod.Spec = cj.Spec.JobTemplate.Spec.Template.Spec
	default:
		log.V(4).Info("no resource found", "got resource", gvr.Resource)
	}
	return pod, nil
}

// 集群污点与容忍
func clusterTolerateFilter(taints []v1.Taint, tolerations []v1.Toleration) bool {
	log := log.FromContext(context.Background())
	filterPredicate := func(t *v1.Taint) bool {
		// PodToleratesNodeTaints is only interested in NoSchedule and NoExecute taints.
		return t.Effect == v1.TaintEffectNoSchedule || t.Effect == v1.TaintEffectNoExecute
	}

	taint, isUntolerated := v1helper.FindMatchingUntoleratedTaint(taints, tolerations, filterPredicate)
	if !isUntolerated {
		return true
	}
	log.V(4).Info("clusterhad taint, that the application didn't tolerate",
		"key", taint.Key, "value", taint.Value)
	return false
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
