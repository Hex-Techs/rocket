package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/manager"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cerificateauthority"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/models/module"
	"github.com/hex-techs/rocket/pkg/models/project"
	"github.com/hex-techs/rocket/pkg/models/publicresource"
	"github.com/hex-techs/rocket/pkg/models/revisionmanager"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	flag "github.com/spf13/pflag"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultTimeout = 1 * time.Second

func main() {
	var configFile string
	flag.StringVarP(&configFile, "config", "c", "./config.yaml", "config file path")
	flag.Parse()
	dir := filepath.Dir(configFile)
	fileName := filepath.Base(configFile)
	cache.InitCommunicationManager()
	r := initServer(dir, fileName)
	run(r)
}

func run(r *gin.Engine) {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Read().Manager.ServerPort),
		Handler: r,
	}
	go func() {
		log.Infof("start manager on :%d", config.Read().Manager.ServerPort)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("start manager with error: %s\n", err)
		}
	}()
	time.Sleep(1 * time.Second)
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Infof("timeout of %v.", defaultTimeout)
	}
	log.Info("Server exiting")
}

func initServer(dir, name string) *gin.Engine {
	if err := config.Load(dir, name, false); err != nil {
		panic(err)
	}

	logger := log.InitLogger()
	defer logger.Sync()
	s := storage.NewEngine(config.Read().DB.Host, config.Read().DB.DB, config.Read().DB.User, config.Read().DB.Password)
	initDB(s)
	fakerDate(s)
	var r *gin.Engine
	if config.Read().Log.Level != "trace" {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	} else {
		r = gin.Default()
	}
	manager.InstallAPI(r, s)
	return r
}

func initDB(s *storage.Engine) {
	log.Info("initializing database...")
	db := s.Client().(*gorm.DB)
	// 设置gorm日志模式
	if config.Read().DB.SqlDebug {
		db.Config.Logger = logger.Default.LogMode(logger.Info)
	} else {
		db.Config.Logger = logger.Default.LogMode(logger.Warn)
	}
	// 设置连接池
	if config.Read().DB.MaxIdleConns != 0 {
		c, _ := db.DB()
		c.SetMaxIdleConns(config.Read().DB.MaxIdleConns)
	}
	if config.Read().DB.MaxOpenConns != 0 {
		c, _ := db.DB()
		c.SetMaxOpenConns(config.Read().DB.MaxOpenConns)
	}
	// migrate
	if err := db.AutoMigrate(&cerificateauthority.User{}, &project.Project{},
		&module.Module{}, &revisionmanager.RevisionManager{}, &publicresource.PublicResource{},
		&cluster.Cluster{}, &application.Application{}); err != nil {
		log.Fatalf("auto migrate table error: %v", err)
	}
	log.Info("initialize database success")
}

func fakerDate(s *storage.Engine) {
	if config.Read().FakerDate {
		prs := publicresource.FakeData(5)
		if err := s.Create(context.Background(), &prs); err != nil {
			log.Warnf("create public resource error: %v", err)
		}

		cas := cerificateauthority.FakeData(10)
		if err := s.Create(context.Background(), &cas); err != nil {
			log.Warnf("create certificate authority error: %v", err)
		}
	}
}
