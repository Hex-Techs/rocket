package main

import (
	"context"
	"fmt"
	"os"

	cli "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	var namespace string
	var name string
	var t string
	var cluster string
	// var approve bool
	pflag.StringVar(&namespace, "namespace", "", "The namespace of workload")
	pflag.StringVar(&name, "name", "", "The name of workload")
	pflag.StringVar(&t, "type", "Scheduling", "The schedule type of workload")
	pflag.StringVar(&cluster, "cluster", "", "scheduler to this cluster")
	pflag.Parse()

	if name == "" {
		fmt.Printf("'--name' must be set")
		os.Exit(1)
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		panic(err)
	}
	cli := cli.NewForConfigOrDie(config)
	workload, err := cli.RocketV1alpha1().Workloads(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	if cluster != "" {
		workload.Status.Clusters = []string{cluster}
		workload.Status.Phase = "Running"
	} else {
		workload.Status.Phase = t
	}
	_, err = cli.RocketV1alpha1().Workloads(namespace).UpdateStatus(context.TODO(), workload, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}
}
