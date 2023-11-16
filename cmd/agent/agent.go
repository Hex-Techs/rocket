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
	"path/filepath"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent"

	// "github.com/hex-techs/rocket/pkg/agent/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	kruiseapi "github.com/openkruise/kruise-api"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kruiseapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	var enabledSchemes []string
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var configFile string
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8090", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8091", "The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.StringSliceVar(&enabledSchemes, "enabled-schemes", []string{}, "The schemes enabled in this agent cluster.")
	pflag.StringVarP(&configFile, "config", "c", "./config.yaml", "config file path")
	pflag.Parse()

	dir := filepath.Dir(configFile)
	fileName := filepath.Base(configFile)
	if err := config.Load(dir, fileName, false); err != nil {
		panic(err)
	}

	logger := log.InitLogger()
	defer logger.Sync()
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
		log.Fatalf("unable to start agent: %v", err)
	}

	if err := agent.InitReconcilers(mgr, enabledSchemes); err != nil {
		log.Fatalf("unable to create controllers: %v", err)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatalf("unable to set up health check: %v", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatalf("unable to set up ready check: %v", err)
	}

	log.Info("starting agent manager")
	// go func() {
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("problem running manager: %v", err)
	}
	// }()
	// for {
	// 	_, ok := <-mgr.Elected()
	// 	if !ok {
	// 		break
	// 	}
	// 	time.Sleep(2 * time.Second)
	// }
	// log.Info("starting remote manager")
	// if err := agent.RemoteController(mgr); err != nil {
	// 	log.Fatalf("problem running remote controller: %v", err)
	// }
}
