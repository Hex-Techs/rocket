package modules

import (
	"time"

	"github.com/hex-techs/rocket/pkg/models/module"
	"github.com/hex-techs/rocket/pkg/rocketctl"
	"github.com/hex-techs/rocket/pkg/utils/display"
	"github.com/hex-techs/rocket/pkg/utils/web"
	"github.com/jedib0t/go-pretty/v6/table"
)

func List(filename string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var mItems []module.Module
	if err := cli.List(&web.Request{Page: 1, Limit: -1}, &mItems); err != nil {
		return err
	}
	d := display.NewTerminalDisplay("", 0)
	header := table.Row{"id", "name", "display-name", "parentID", "level", "full-name", "create"}
	var content [][]interface{}
	for _, c := range mItems {
		content = append(content, []interface{}{c.ID, c.Name, c.DisplayName, c.ParentID, c.Level, c.FullName,
			c.CreatedAt.Local().Format(time.DateTime)})
	}
	d.Table(header, content...)
	return nil
}

func Create(filename string, m *module.Module) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Create(m)
}

func Delete(filename, id string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Delete(id, &module.Module{})
}
