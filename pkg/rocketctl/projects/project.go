package projects

import (
	"time"

	"github.com/hex-techs/rocket/pkg/models/project"
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
	var pItems []project.Project
	if err := cli.List(&web.Request{Page: 1, Limit: -1}, &pItems); err != nil {
		return err
	}
	d := display.NewTerminalDisplay("", 0)
	header := table.Row{"id", "name", "display-name", "module", "create"}
	var content [][]interface{}
	for _, c := range pItems {
		content = append(content, []interface{}{c.ID, c.Name, c.DisplayName, c.Module,
			c.CreatedAt.Local().Format(time.DateTime)})
	}
	d.Table(header, content...)
	return nil
}

func Create(filename string, p *project.Project) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Create(p)
}

func Delete(filename, id string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Delete(id, &project.Project{})
}
