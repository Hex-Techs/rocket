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
	pflag.StringVar(&namespace, "namespace", "", "The namespace of application")
	pflag.StringVar(&name, "name", "", "The name of application")
	pflag.StringVar(&t, "type", "Scheduling", "The schedule type of application")
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
	application, err := cli.RocketV1alpha1().Applications(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	if cluster != "" {
		application.Status.Clusters = []string{cluster}
		application.Status.Phase = "Running"
	} else {
		application.Status.Phase = t
	}
	_, err = cli.RocketV1alpha1().Applications(namespace).UpdateStatus(context.TODO(), application, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}
}
