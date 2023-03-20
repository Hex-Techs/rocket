package main

import (
	"context"
	"fmt"
	"os"

	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	var name string
	var approve bool
	pflag.StringVar(&name, "name", "", "The name of cluster")
	pflag.BoolVar(&approve, "approve", true, "Approve the cluster")
	pflag.Parse()

	if name == "" {
		fmt.Printf("'--name' must be set")
		os.Exit(1)
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		panic(err)
	}
	cli := clientset.NewForConfigOrDie(config)
	cluster, err := cli.RocketV1alpha1().Clusters().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("%s is not found", name)
		}
		panic(err)
	}
	cluster.Status.State = "Approved"
	_, err = cli.RocketV1alpha1().Clusters().UpdateStatus(context.TODO(), cluster, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("approve cluster with error: %v", err)
	}
}
