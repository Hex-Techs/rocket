package cerificateauthority

import (
	"github.com/hex-techs/rocket/pkg/utils/config"
)

func Login(path, addr, uname, passwd string) error {
	cfg := &config.CmdConfig{
		ManagerAddress: addr,
		Username:       uname,
		AuthKey:        []byte(passwd),
	}
	return config.WriteToFile(path, cfg)
}
