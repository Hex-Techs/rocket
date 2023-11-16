package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/fize/go-ext/log"
	agentapp "github.com/hex-techs/rocket/pkg/agent/application"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/scheduler/filter"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"github.com/hex-techs/rocket/pkg/utils/web"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
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

func (r *SchedulerReconciler) scheduler(ctx context.Context, application *application.Application) []ClusterResult {
	if len(r.schedCache) == 0 {
		log.Info("no cluster can be scheduled", "application", tools.KObj(application))
	}
	var scoreMap = []ClusterResult{}
	regions := mapset.NewSet[string]()
	for i := range application.Regions {
		regions.Add(application.Regions[i])
	}
	area, err := r.areaEnv(application.Name, application.Namespace)
	if err != nil {
		log.Errorw("unable to scheduler application", "application", tools.KObj(application), "error", err)
		return scoreMap
	}
	p, err := newPod(ctx, application)
	if err != nil {
		log.Errorw("unable to scheduler application", "application", tools.KObj(application), "error", err)
		return scoreMap
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	log.Debug("Attempting to schedule application", "application", tools.KObj(application))
	for _, cluster := range r.schedCache {
		if !clusterTolerateFilter(cluster.Taint, r.generateTolerations(application)) {
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
	log.Debug("scheduler application in selected", "application", tools.KObj(application), "allScore", scoreMap)
	scoreMap = r.removeNotMatchCluster(application, scoreMap)
	s := sortResult{d: scoreMap}
	sort.Sort(s)
	log.Debug("scheduler application done", "application", tools.KObj(application), "resultScore", s.d)
	return s.d
}

// TODO: ignore toleration for application just now
func (r *SchedulerReconciler) generateTolerations(app *application.Application) []v1.Toleration {
	tolerations := []v1.Toleration{}
	return tolerations
}

func (r *SchedulerReconciler) areaEnv(name, namespace string) (area string, err error) {
	app := application.Application{}
	err = r.Get(name, &web.Request{Namespace: namespace}, &app)
	if err != nil {
		return "", err
	}
	if app.CloudArea == "" {
		return "", fmt.Errorf("%s cloud area is empty", name)
	}
	if app.Environment == "" {
		return "", fmt.Errorf("%s environment is empty", name)
	}
	return app.CloudArea, nil
}

func (r *SchedulerReconciler) scheddulePod(ctx context.Context, clustername string, pod *v1.Pod) (result *ClusterResult, err error) {
	if _, ok := r.snapshot[clustername]; !ok {
		return result, fmt.Errorf("cluster %s not found in snapshot, will retry", clustername)
	}
	if err := r.schedCache[clustername].schedCache.UpdateSnapshot(r.snapshot[clustername]); err != nil {
		return result, err
	}
	log.Debug("Snapshotting scheduler cache and node infos done")
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
	log.Infow("Pod scheduler", "Pod", pod)
	for _, n := range allNodes {
		if filter.Filter(ctx, pod, n) {
			result.Score = result.Score + filter.Score(ctx, pod, n)
		}
	}
	return result, nil
}

func (r *SchedulerReconciler) removeNotMatchCluster(application *application.Application, scoreMap []ClusterResult) []ClusterResult {
	desired := 1
	// TODO: 设置 replicas
	// replicas := application.Replicas
	// if replicas != nil {
	// 	desired = int(*replicas)
	// }
	result := []ClusterResult{}
	// 将不符合replicas要求的集群剔除
	for _, cluster := range scoreMap {
		if cluster.Score >= int64(desired) {
			result = append(result, cluster)
		}
	}
	return result
}

func newPod(ctx context.Context, application *application.Application) (*v1.Pod, error) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      application.Name,
			Namespace: application.Namespace,
		},
		Spec: v1.PodSpec{},
	}
	gvr, resource := agentapp.GenerateWorkload(application)
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
		log.Infow("no resource found", "got resource", gvr.Resource)
	}
	return pod, nil
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
	log.Infow("clusterhad taint, that the application didn't tolerate",
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
