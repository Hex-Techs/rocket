package casbin

import (
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	ext "github.com/fize/go-ext/config"
	"github.com/fize/go-ext/log"
	"github.com/hex-techs/blade/pkg/utils/config"
)

func Init() (*casbin.Enforcer, error) {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, mod, obj, act")           // request definition
	m.AddDef("p", "p", "sub, mod, obj, act")           // policy definition
	m.AddDef("g", "g", "_, _")                         // role definition
	m.AddDef("e", "e", "some(where (p.eft == allow))") // effect
	m.AddDef("m", "m", "g(r.sub, p.sub, r.mod) && r.mod == p.mod && r.obj == p.obj && r.act == p.act")

	var a *gormadapter.Adapter
	var err error

	if config.Read().DB.Type == ext.Mysql {
		link := fmt.Sprintf("%s:%s@tcp(%s)/",
			config.Read().DB.User, config.Read().DB.Password,
			config.Read().DB.Host)
		for {
			a, err = gormadapter.NewAdapter("mysql", link, config.Read().DB, "casbin")
			if err != nil {
				log.Error(err)
				time.Sleep(5 * time.Second)
			} else {
				break
			}
		}
	} else {
		a, err = gormadapter.NewAdapter("sqlite3", config.Read().DB.DB)
		if err != nil {
			return nil, err
		}
	}
	return casbin.NewEnforcer(m, a)
}
