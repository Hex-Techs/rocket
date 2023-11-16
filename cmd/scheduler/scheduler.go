package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	"github.com/hex-techs/rocket/pkg/scheduler"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	next := make(chan struct{})
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go communication.InitCommunication(next)
	<-next
	log.Infow("start scheduler")
	sched := scheduler.NewReconciler()
	ctx := context.Background()
	var cancel context.CancelFunc
	go sched.Run(ctx)
	<-quit
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Info("timeout")
	}
	log.Info("scheduler exit")
}
