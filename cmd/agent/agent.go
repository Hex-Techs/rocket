/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	"github.com/hex-techs/rocket/pkg/agent"
	"github.com/hex-techs/rocket/pkg/agent/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	kruiseapi "github.com/openkruise/kruise-api"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kruiseapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	param := new(config.CommandParam)
	kruiseParam := new(config.OpenKruise)
	kedaParam := new(config.Keda)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	var enabledSchemes []string
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8090", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8091", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.StringSliceVar(&enabledSchemes, "enabled-schemes", []string{""}, "The schemes enabled in this agent cluster.")

	pflag.StringVar(&param.Name, "name", "", "The name of this agent cluster.")
	pflag.StringVar(&param.Region, "region", "beijing", "The region of this agent cluster.")
	pflag.StringVar(&param.Area, "area", "public", "The area of this agent cluster.")
	pflag.StringVar(&param.Environment, "environment", "test", "The environment of this agent cluster.")
	pflag.StringVar(&param.MasterURL, "master-url", "", "The manager cluster apiserver url.")
	pflag.StringVar(&param.BootstrapToken, "bootstrap-token", "", "The bootstrap token use to register.")
	pflag.IntVar(&param.KeepAliveSecond, "keepalive-second", 300, "The heartbeet interval time, default: 300.")

	pflag.BoolVar(&kruiseParam.Enabled, "kruise-enabled", true, "enabled openKruise.")
	pflag.StringVar(&kruiseParam.Repository, "kruise-image", "openkruise/kruise-manager", "open-kruise image repository")
	pflag.StringVar(&kruiseParam.Tag, "kruise-image-tag", "v1.4.0", "open-kruise image tag")
	pflag.StringVar(&kruiseParam.RequestCPU, "kruise-request-cpu", "200m", "open-kruise request cpu")
	pflag.StringVar(&kruiseParam.LimitCPU, "kruise-limit-cpu", "1", "open-kruise limit cpu")
	pflag.StringVar(&kruiseParam.RequestMem, "kruise-request-mem", "256Mi", "open-kruise request memory")
	pflag.StringVar(&kruiseParam.LimitMem, "kruise-limit-mem", "1Gi", "open-kruise limit memory")
	pflag.StringVar(&kruiseParam.FeatureGates, "kruise-feature-gates",
		"WorkloadSpread=true,PreDownloadImageForInPlaceUpdate=true,CloneSetPartitionRollback=true,ResourcesDeletionProtection=true,PodUnavailableBudgetDeleteGate=true,PodUnavailableBudgetUpdateGate=true",
		"open-kruise feature-gates")

	pflag.BoolVar(&kedaParam.Enabled, "keda-enabled", true, "enabled keda.")
	pflag.StringVar(&kedaParam.Repository, "keda-image", "ghcr.io/kedacore/keda", "keda image repository")
	pflag.StringVar(&kedaParam.Tag, "keda-image-tag", "2.7.1", "keda image tag")
	pflag.StringVar(&kedaParam.MetricRepository, "keda-metrics-image", "ghcr.io/kedacore/keda-metrics-apiserver", "keda metrics apiserver image repository")
	pflag.StringVar(&kedaParam.MetricTag, "keda-metrics-tag", "2.7.1", "keda metrics apiserver image tag")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(klogr.New())

	config.Set(param)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "74edaf61.hextech.io",
		LeaderElectionNamespace: "rocket-system",
	})
	if err != nil {
		setupLog.Error(err, "unable to start agent")
		os.Exit(1)
	}
	moduleParam := &config.ModuleParams{
		Kruise: kruiseParam,
		Keda:   kedaParam,
	}
	cluster.RegisterInit(param, mgr, moduleParam)

	if err := agent.InitReconcilers(mgr, enabledSchemes); err != nil {
		setupLog.Error(err, "unable to create controllers")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()
	// 成为 leader 之后启动 controller
	for {
		_, ok := <-mgr.Elected()
		if !ok {
			break
		}
		time.Sleep(2 * time.Second)
	}
	klog.Info("starting remote manager")
	if err := agent.RemoteController(mgr); err != nil {
		setupLog.Error(err, "problem running remote controller")
		os.Exit(1)
	}
}
