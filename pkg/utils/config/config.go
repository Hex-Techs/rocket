package config

import (
	"fmt"
	"sync"

	ext "github.com/fize/go-ext/config"
	"github.com/kkyr/fig"
)

// manager configuration
type ManagerServiceConfig struct {
	ServerPort int `fig:"serverPort" default:"9100"`
	// reset password use this domain
	Domain string `fig:"domain"`
	// internal agent use this address
	InternalAddress     string   `fig:"internalAddress" default:"http://127.0.0.1:9100"`
	TokenExpired        int64    `fig:"tokenExpired" default:"86400"`
	URLExpired          int64    `fig:"urlExpired" default:"600"`
	ResetPath           string   `fig:"resetPath"`
	Cors                bool     `fig:"cors"`
	AllowOrigin         []string `fig:"allowOrigin"`
	CompanyPost         string   `fig:"company"`
	APIDoc              bool     `fig:"apiDoc"`
	AgentBootstrapToken string   `fig:"agentBootstrapToken"`
}

func (m *ManagerServiceConfig) validate() error {
	if m.AgentBootstrapToken == "" {
		return fmt.Errorf("agent agentBootstrapToken is empty")
	}
	return nil
}

// agent configuration
type AgentServiceConfig struct {
	Name            string `fig:"name"`
	Region          string `fig:"region"`
	Area            string `fig:"area" default:"public"`
	ManagerAddress  string `fig:"managerAddress" default:"http://127.0.0.1:9100"`
	BootstrapToken  string `fig:"bootstrapToken"`
	KeepAliveSecond int    `fig:"keepAliveSecond" default:"300"`
	// agent mode, support: hub, edge
	Mode string `fig:"mode" default:"edge"`
}

func (a *AgentServiceConfig) validate() error {
	if a.Name == "" {
		return fmt.Errorf("agent name is empty")
	}
	if a.Region == "" {
		return fmt.Errorf("agent region is empty")
	}
	if a.Area == "" {
		return fmt.Errorf("agent area is empty")
	}
	if a.ManagerAddress == "" {
		return fmt.Errorf("agent managerAddress is empty")
	}
	if a.BootstrapToken == "" {
		return fmt.Errorf("agent bootstrapToken is empty")
	}
	return nil
}

// SchedulerConfig configuration
type SchedulerConfig struct {
	ManagerAddress string `fig:"managerAddress" default:"http://127.0.0.1:9100"`
	BootstrapToken string `fig:"bootstrapToken"`
}

// ldap
type Ldap struct {
	Enabled bool `fig:"enabled" default:"false"`
	// 127.0.0.1:389
	Host string `fig:"host" default:"127.0.0.1:389"`
	// baseDN ou=Users,dc=blade,dc=cn
	BaseDN string `fig:"baseDN"`
	// cn=admin,dc=blade,dc=cn
	User     string `fig:"user"`
	Password string `fig:"password"`
}

// Config
type Config struct {
	ext.Config `fig:",squash"`
	// manager
	Manager *ManagerServiceConfig `fig:"manager"`
	// agent
	Agent *AgentServiceConfig `fig:"agent"`
	// scheduler
	Scheduler *SchedulerConfig `fig:"scheduler"`
	// ldap
	Ldap *Ldap `fig:"ldap"`
	// generate fake data
	FakerDate bool `fig:"fakerDate"`
}

var (
	lock   *sync.RWMutex
	config *Config
)

// Load load config, only call once
func Load(dir, name string, agent bool) error {
	lock = new(sync.RWMutex)
	lock.Lock()
	defer lock.Unlock()
	config = new(Config)

	if err := ext.Load(dir, name); err != nil {
		return fmt.Errorf("load base config error: %s", err)
	}

	if err := fig.Load(config, fig.Dirs(dir), fig.File(name)); err != nil {
		return err
	}

	if agent {
		if config.Agent == nil {
			config.Agent = new(AgentServiceConfig)
		}
		if err := config.Agent.validate(); err != nil {
			return err
		}
		if config.Agent.Mode == "" {
			config.Agent.Mode = "edge"
		}
		if config.Agent.Mode != "edge" && config.Agent.Mode != "hub" {
			return fmt.Errorf("agent mode only support edge and hub")
		}
	} else {
		if config.Manager == nil {
			config.Manager = new(ManagerServiceConfig)
		}
		if err := config.Manager.validate(); err != nil {
			return err
		}
		config.DB = ext.Read().DB
		config.Email = ext.Read().Email
		config.Ldap = new(Ldap)
	}
	return nil
}

// Read
func Read() *Config {
	return config
}
