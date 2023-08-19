package config

import (
	"fmt"

	zaplog "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	param   *CommandParam
	LogOpts = zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
		ZapOpts:     []zaplog.Option{zaplog.AddCaller()},
	}
)

// 命令行参数选项
type CommandParam struct {
	// 名称
	Name string
	// 地域
	Region string
	// 云环境
	Area string
	// master url
	MasterURL string
	// bootstrap token
	BootstrapToken string
	// 心跳周期
	KeepAliveSecond int
	// 超时时间
	TimeOut int
}

func (cp *CommandParam) Required() error {
	if len(cp.Name) == 0 {
		return fmt.Errorf("'name' must be specified")
	}
	if len(cp.BootstrapToken) == 0 {
		return fmt.Errorf("'bootstrap-token' must be specified")
	}
	return nil
}

func Set(p *CommandParam) {
	param = p
}

func Pread() *CommandParam {
	return param
}
