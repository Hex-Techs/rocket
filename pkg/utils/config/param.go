package config

import "fmt"

var param *CommandParam

// 命令行参数选项
type CommandParam struct {
	// 名称
	Name string
	// 地域
	Region string
	// 云环境
	Area string
	// 环境
	Environment string
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
