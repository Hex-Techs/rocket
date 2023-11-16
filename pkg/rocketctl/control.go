package rocketctl

import (
	"github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/utils/config"
)

func InitClient(filename string) (client.Interface, error) {
	cfg, err := config.GetFromFile(filename)
	if err != nil {
		return nil, err
	}
	return client.New(cfg.ManagerAddress, string(cfg.AuthKey)), nil
}
